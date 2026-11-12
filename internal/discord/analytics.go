package discord

import (
	"LazyCatBot/internal/sirus"
	"LazyCatBot/internal/storage"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var classNames = map[int]string{
	1:  "Воин",
	2:  "Паладин",
	3:  "Охотник",
	4:  "Разбойник",
	5:  "Жрец",
	6:  "Рыцарь смерти",
	7:  "Шаман",
	8:  "Маг",
	9:  "Чернокнижник",
	11: "Друид",
}

func className(id int) string {
	if n, ok := classNames[id]; ok {
		return n
	}
	return "Неизвестно"
}

func formatProfileTime(unix int64) string {
	if unix == 0 {
		return "—"
	}
	return time.Unix(unix, 0).In(time.FixedZone("MSK", 3*3600)).Format("02.01.2006 15:04")
}

// requirePremium replies with an upsell and returns false when the channel has
// no active premium entitlement.
func (h *BotHandler) requirePremium(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	if h.PremiumStore != nil && h.PremiumStore.IsPremium(i.ChannelID) {
		return true
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "🔒 Эта функция доступна с премиумом для канала.\n" +
				"Оформите его командой `/premium` — код активации выдаётся сразу.",
			Flags: discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{Label: "🐾 Оформить премиум", Style: discordgo.PrimaryButton, CustomID: "premium_open"},
				}},
			},
		},
	})
	return false
}

// firstGuildSub replies with an error and returns false when the channel has no
// tracked guild.
func (h *BotHandler) firstGuildSub(s *discordgo.Session, i *discordgo.InteractionCreate) (storage.GuildSub, bool) {
	subs, err := h.SubStore.GetGuildSubsByChannel(i.ChannelID)
	if err != nil || len(subs) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ В этом канале не настроено отслеживание гильдий.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return storage.GuildSub{}, false
	}
	return subs[0], true
}

func (h *BotHandler) HandleAttendanceCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !h.requirePremium(s, i) {
		return
	}
	if h.KillHistory == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ История киллов ещё не подключена.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	sub, ok := h.firstGuildSub(s, i)
	if !ok {
		return
	}

	since := time.Now().AddDate(0, 0, -28).Unix()
	total, entries, err := h.KillHistory.GetAttendance(sub.Realm, sub.GuildID, since)
	if err != nil {
		h.Logger.Error("GetAttendance error", "error", err, "guild_id", sub.GuildID, "realm", sub.Realm)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Не удалось получить данные о посещаемости.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	if total == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "📭 За последние 4 КД киллов не найдено. История собирается с момента обновления бота.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	guildName := "гильдии"
	if n, err := h.SubStore.GetGuildName(sub.GuildID, sub.Realm); err == nil && n != "" {
		guildName = n
	}

	limit := len(entries)
	if limit > 15 {
		limit = 15
	}
	var b strings.Builder
	for idx := 0; idx < limit; idx++ {
		e := entries[idx]
		percent := e.Kills * 100 / total
		if percent > 100 {
			percent = 100
		}
		fmt.Fprintf(&b, "`%2d.` **%s** — %d/%d (%d%%)\n", idx+1, e.Name, e.Kills, total, percent)
	}
	if len(entries) > limit {
		fmt.Fprintf(&b, "…и ещё %d игроков", len(entries)-limit)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "📋 Посещаемость — " + guildName,
		Description: fmt.Sprintf("Период: **4 КД** · киллов: **%d** · игроков: **%d**", total, len(entries)),
		Color:       0xf1c40f,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Явка", Value: b.String()},
		},
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
}

func (h *BotHandler) HandleProfileCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !h.requirePremium(s, i) {
		return
	}
	if h.KillHistory == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ История киллов ещё не подключена.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	data := i.ApplicationCommandData()
	name := ""
	for _, o := range data.Options {
		if o.Name == "name" {
			name = strings.TrimSpace(o.StringValue())
		}
	}
	if name == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Укажите ник персонажа.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	sub, ok := h.firstGuildSub(s, i)
	if !ok {
		return
	}

	since := time.Now().AddDate(0, 0, -28).Unix()
	p, err := h.KillHistory.GetPlayerProfile(sub.Realm, sub.GuildID, name, since)
	if err != nil {
		h.Logger.Error("GetPlayerProfile error", "error", err, "player", name, "guild_id", sub.GuildID)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Не удалось получить профиль игрока.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	if p.TotalKills == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("📭 По игроку **%s** нет киллов за последние 4 КД.", name),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	role := "ДД"
	if p.Role == "hps" {
		role = "Хил"
	}
	fields := []*discordgo.MessageEmbedField{
		{Name: "Класс и спек", Value: fmt.Sprintf("%s · %s", className(p.ClassID), sirus.GetSpecName(p.ClassID, p.SpecID)), Inline: true},
		{Name: "Роль", Value: role, Inline: true},
		{Name: "Илвл", Value: fmt.Sprintf("%d", p.Ilvl), Inline: true},
		{Name: "Посещаемость", Value: fmt.Sprintf("%d%% (%d килов)", p.AttendedPct, p.TotalKills), Inline: true},
		{Name: "Лучший DPS", Value: FormatNum(p.BestDps), Inline: true},
		{Name: "Лучший HPS", Value: FormatNum(p.BestHps), Inline: true},
		{Name: "Первый килл в окне", Value: formatProfileTime(p.FirstSeen), Inline: true},
		{Name: "Последний килл", Value: formatProfileTime(p.LastSeen), Inline: true},
	}

	embed := &discordgo.MessageEmbed{
		Title:  "👤 Профиль — " + p.Name,
		Color:  0xf1c40f,
		Fields: fields,
		Footer: &discordgo.MessageEmbedFooter{Text: "Период: последние 4 КД"},
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
}
