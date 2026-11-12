package discord

import (
	"LazyCatBot/internal/storage"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// PremiumPricing is shown on the /premium screen. Prices are not enforced by
// the bot: the donation provider reports the amount, and the code carries the
// plan. They exist so the instructions can tell the donor what to pay.
type PremiumPricing struct {
	MonthlyUSD  int
	YearlyUSD   int
	LifetimeUSD int
}

func (p PremiumPricing) price(plan storage.PremiumPlan) int {
	switch plan {
	case storage.PlanMonthly:
		return p.MonthlyUSD
	case storage.PlanYearly:
		return p.YearlyUSD
	case storage.PlanLifetime:
		return p.LifetimeUSD
	}
	return 0
}

func premiumPlanBySuffix(s string) (storage.PremiumPlan, bool) {
	switch s {
	case string(storage.PlanMonthly):
		return storage.PlanMonthly, true
	case string(storage.PlanYearly):
		return storage.PlanYearly, true
	case string(storage.PlanLifetime):
		return storage.PlanLifetime, true
	}
	return "", false
}

func formatPremiumExpiry(e *storage.PremiumEntitlement) string {
	if e == nil {
		return "—"
	}
	if e.Lifetime() {
		return "бессрочно"
	}
	loc := time.FixedZone("MSK", 3*3600)
	return e.ExpiresAtTime().In(loc).Format("02.01.2006 15:04")
}

func hasManageChannels(i *discordgo.InteractionCreate) bool {
	if i.Member == nil {
		return false
	}
	return i.Member.Permissions&discordgo.PermissionManageChannels != 0
}

func premiumDenied(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "⛔ Управление премиумом доступно участникам с правом «Управление каналами».",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// HandlePremiumCommand shows the current premium status of the channel and the
// purchase buttons. Premium is per channel: buying in one channel does not
// enable another.
func (h *BotHandler) HandlePremiumCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !hasManageChannels(i) {
		premiumDenied(s, i)
		return
	}
	if h.PremiumStore == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Премиум ещё не подключён на этом инстансе.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	ent, err := h.PremiumStore.Get(i.ChannelID)
	if err != nil {
		h.Logger.Error("Get premium entitlement error", "error", err, "channel_id", i.ChannelID)
	}

	var status string
	now := time.Now()
	switch {
	case ent == nil:
		status = "Премиум в этом канале **не активирован**."
	case ent.Active(now) && (ent.Lifetime() || ent.ExpiresAt >= now.Unix()):
		status = fmt.Sprintf("✅ Премиум активен: **%s**, действует до **%s**.",
			ent.Plan.Title(), formatPremiumExpiry(ent))
	case ent.Active(now):
		status = fmt.Sprintf("⚠️ Премиум истёк (%s), но ещё действует льготный период 72 ч. Продлите подписку.",
			formatPremiumExpiry(ent))
	default:
		status = fmt.Sprintf("⏳ Премиум истёк (%s). Продлите подписку, чтобы не потерять функции.",
			formatPremiumExpiry(ent))
	}

	desc := "**🐾 Премиум для этого канала**\n" + status + "\n\n" +
		"Что входит:\n" +
		"— отчёты без очереди в пик рейдов;\n" +
		"— рейтинги за 4 КД и за всё время;\n" +
		"— приоритетная поддержка.\n\n" +
		"Новые аналитические функции добавляются по мере выхода.\n" +
		"Премиум привязан к каналу: остальные каналы сервера остаются бесплатными."

	buttons := []discordgo.MessageComponent{}
	for _, plan := range []storage.PremiumPlan{storage.PlanMonthly, storage.PlanYearly, storage.PlanLifetime} {
		label := plan.Title()
		if price := h.PremiumPricing.price(plan); price > 0 {
			label = fmt.Sprintf("%s — $%d", label, price)
		}
		buttons = append(buttons, discordgo.Button{
			Label:    label,
			Style:    discordgo.PrimaryButton,
			CustomID: "premium_buy_" + string(plan),
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{Title: "🐾 LazyCatBot Premium", Description: desc, Color: 0xf1c40f},
			},
			Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: buttons}},
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

// HandlePremiumButton creates a one-time activation code for the selected plan
// and returns step-by-step instructions. It returns true when the interaction
// was handled.
func (h *BotHandler) HandlePremiumButton(s *discordgo.Session, i *discordgo.InteractionCreate, customID string) bool {
	suffix, ok := strings.CutPrefix(customID, "premium_buy_")
	if !ok {
		return false
	}
	plan, ok := premiumPlanBySuffix(suffix)
	if !ok {
		return false
	}
	if h.PremiumStore == nil {
		return false
	}
	if !hasManageChannels(i) {
		premiumDenied(s, i)
		return true
	}

	userID := ""
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	code, expires, err := h.PremiumStore.CreateCode(i.ChannelID, userID, plan)
	if err != nil {
		h.Logger.Error("Create premium code error", "error", err, "channel_id", i.ChannelID, "plan", plan)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Не удалось создать код активации. Попробуйте позже.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return true
	}

	price := h.PremiumPricing.price(plan)
	priceLine := ""
	if price > 0 {
		priceLine = fmt.Sprintf("**$%d**", price)
	} else {
		priceLine = "сумму, указанную на странице доната"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "🎟 Код активации: **%s**\n\n", code)
	b.WriteString("**Как активировать премиум:**\n")
	b.WriteString("1. Откройте страницу доната")
	if h.Donation.URL != "" {
		fmt.Fprintf(&b, ": <%s>", h.Donation.URL)
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "2. В поле «Сообщение» вставьте код **%s**.\n", code)
	b.WriteString("   Можно добавить свой игровой ник — он получит премиум-подсветку в отчётах.\n")
	fmt.Fprintf(&b, "3. Отправьте донат: %s (план «%s»).\n", priceLine, plan.Title())
	fmt.Fprintf(&b, "4. Бот активирует премиум в канале <#%s> автоматически в течение нескольких минут.\n\n", i.ChannelID)
	fmt.Fprintf(&b, "Код одноразовый и действует до **%s** (МСК). Не передавайте его другим.\n",
		expires.In(time.FixedZone("MSK", 3*3600)).Format("02.01.2006 15:04"))

	content := b.String()
	if h.Donation.URL == "" {
		content += "\n⚠️ Оплата на этом инстансе ещё не настроена — сообщите администратору бота."
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	return true
}
