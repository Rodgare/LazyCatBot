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
	MonthlyUSD int
}

func (p PremiumPricing) price(plan storage.PremiumPlan) int {
	if plan == storage.PlanMonthly {
		return p.MonthlyUSD
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

// HandlePremiumCommand shows the current premium status of the server and the
// purchase buttons. Premium is per Discord server: buying once unlocks every
// channel of the server.
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

	ent, err := h.PremiumStore.Get(i.GuildID)
	if err != nil {
		h.Logger.Error("Get premium entitlement error", "error", err, "guild_id", i.GuildID)
	}

	var status string
	now := time.Now()
	switch {
	case ent == nil:
		status = "Премиум на этом сервере **не активирован**."
	case ent.Active(now) && (ent.Lifetime() || ent.ExpiresAt >= now.Unix()):
		status = fmt.Sprintf("✅ Премиум активен для всего сервера: **%s**, действует до **%s**.",
			ent.Plan.Title(), formatPremiumExpiry(ent))
	case ent.Active(now):
		status = fmt.Sprintf("⚠️ Премиум истёк (%s), но ещё действует льготный период 72 ч. Продлите подписку.",
			formatPremiumExpiry(ent))
	default:
		status = fmt.Sprintf("⏳ Премиум истёк (%s). Продлите подписку, чтобы не потерять функции.",
			formatPremiumExpiry(ent))
	}

	desc := "**🐾 Премиум для всего сервера**\n" + status + "\n\n" +
		"Что входит:\n" +
		"— отчёты без очереди в пик рейдов во всех каналах;\n" +
		"— рейтинги за 4 КД и за всё время;\n" +
		"— еженедельный дайджест;\n" +
		"— посещаемость и профили игроков;\n" +
		"— приоритетная поддержка.\n\n" +
		fmt.Sprintf("Подписка: **$%d/мес**. Отменить можно в любой момент.\n", h.PremiumPricing.MonthlyUSD) +
		"Премиум действует на весь сервер: оплата один раз открывает все каналы."

	label := storage.PlanMonthly.Title()
	if price := h.PremiumPricing.price(storage.PlanMonthly); price > 0 {
		label = fmt.Sprintf("%s — $%d/мес", label, price)
	}
	buttons := []discordgo.MessageComponent{
		discordgo.Button{
			Label:    label,
			Style:    discordgo.PrimaryButton,
			CustomID: "premium_buy_" + string(storage.PlanMonthly),
		},
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

// HandleGrantPremiumCommand gives every known Discord server an 8-day trial.
// It is restricted to ADMIN_USER_ID. Existing active entitlements are skipped so
// a paid plan is never overwritten by the trial.
func (h *BotHandler) HandleGrantPremiumCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !h.isAdmin(i) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⛔ У вас нет прав на эту команду.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
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

	guildIDs, err := h.SubStore.GetAllDiscordGuildIDs()
	if err != nil {
		h.Logger.Error("List guilds for trial error", "error", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Не удалось получить список серверов.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		h.Logger.Error("Defer trial grant response error", "error", err)
		return
	}

	actor := ""
	if i.Member != nil && i.Member.User != nil {
		actor = i.Member.User.ID
	} else if i.User != nil {
		actor = i.User.ID
	}

	now := time.Now()
	granted, alreadyPremium, failed := 0, 0, 0
	for _, guildID := range guildIDs {
		if ent, err := h.PremiumStore.Get(guildID); err == nil && ent != nil && ent.Active(now) {
			alreadyPremium++
			continue
		}
		if _, err := h.PremiumStore.Grant(guildID, "", storage.PlanTrial, "trial", actor); err != nil {
			failed++
			h.Logger.Error("Grant trial error", "error", err, "guild_id", guildID)
			continue
		}
		granted++
	}

	content := fmt.Sprintf("🎁 Пробный премиум на 8 дней выдан **%d** серверам.", granted)
	if alreadyPremium > 0 {
		content += fmt.Sprintf("\nУже с активным премиумом: **%d** (пропущены).", alreadyPremium)
	}
	if failed > 0 {
		content += fmt.Sprintf("\nОшибок: **%d** — подробности в логе.", failed)
	}
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
		h.Logger.Error("Edit trial grant response error", "error", err)
	}
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

	code, expires, err := h.PremiumStore.CreateCode(i.GuildID, i.ChannelID, userID, plan)
	if err != nil {
		h.Logger.Error("Create premium code error", "error", err, "guild_id", i.GuildID, "channel_id", i.ChannelID, "plan", plan)
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
		priceLine = fmt.Sprintf("**$%d/мес**", price)
	} else {
		priceLine = "сумму подписки"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "🎟 Код активации: **%s**\n\n", code)
	b.WriteString("**Как активировать премиум:**\n")
	b.WriteString("1. Откройте страницу подписки")
	if h.PremiumURL != "" {
		fmt.Fprintf(&b, ": <%s>", h.PremiumURL)
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "2. В поле «Сообщение» вставьте код **%s**.\n", code)
	fmt.Fprintf(&b, "3. Оформите подписку: %s (план «%s»).\n", priceLine, plan.Title())
	fmt.Fprintf(&b, "4. Бот активирует премиум на всём сервере автоматически в течение нескольких минут.\n\n")
	fmt.Fprintf(&b, "Код одноразовый и действует до **%s** (МСК). Не передавайте его другим.\n",
		expires.In(time.FixedZone("MSK", 3*3600)).Format("02.01.2006 15:04"))
	b.WriteString("При продлении Tribute пришлёт повторный платёж — премиум продлится автоматически.\n")

	content := b.String()
	if h.PremiumURL == "" {
		content += "\n⚠️ Страница подписки на этом инстансе не настроена — сообщите администратору бота."
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
