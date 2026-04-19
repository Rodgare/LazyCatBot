package discord

import (
	"fmt"
	"slices"
	"time"

	"github.com/bwmarrin/discordgo"
)

func SendKillReport(s *discordgo.Session, channelID string, report BossKillReport) {
	// 1. Сортировка игроков по ДПС от большего к меньшему
	slices.SortFunc(report.Players, func(a, b PlayerReport) int {
		return b.Dps - a.Dps
	})

	// 2. Формирование таблицы внутри кодового блока
	table := "```text\n"
	table += " №  | Игрок         | ДПС     | Ранг \n"
	table += "----|---------------|---------|------\n"

	for i, p := range report.Players {
		name := p.Name
		if len(name) > 13 {
			name = name[:10] + "..."
		}
		table += fmt.Sprintf("%3d | %-13s | %-7d | #%-3d\n", i+1, name, p.Dps, p.SpecRank)
	}
	table += "```"

	embed := &discordgo.MessageEmbed{
		Title:       "⚔️ " + report.BossName,
		Description: table,
		Color:       0xf1c40f, // Золотой
		Footer: &discordgo.MessageEmbedFooter{
			Text: "LazyCatBot PVE Progression",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	_, err := s.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		fmt.Println("Error sending embed:", err)
	}
}
