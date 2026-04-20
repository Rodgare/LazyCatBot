package discord

import (
	"fmt"
	"slices"
	"time"

	"github.com/bwmarrin/discordgo"
)

func SendKillReport(s *discordgo.Session, channelID string, report BossKillReport) {
	slices.SortFunc(report.Players, func(a, b PlayerReport) int {
		return b.Dps - a.Dps
	})

	table := "```text\n"
	table += " № | Игрок (Спек)  | iLvl | DPS/HPS | Ранг\n"
	table += "---|---------------|------|---------|-----\n"

	for i, p := range report.Players {
		displayName := fmt.Sprintf("%s (%s)", p.Name, p.SpecName)
		if len(displayName) > 13 {
			displayName = displayName[:10] + "..."
		}

		perf := fmt.Sprintf("%d/%d", p.Dps, p.Hps)

		table += fmt.Sprintf("%2d | %-13s | %4d | %-7s | #%-3d\n",
			i+1, displayName, p.Ilvl, perf, p.SpecRank)
	}
	table += "```"

	description := fmt.Sprintf("⏱️ **Время:** %s  |  🔄 **Попытки:** %d\n💥 **Рейд ДПС:** %d\n\n%s",
		report.Duration, report.Attempts, report.TotalDps, table)

	embed := &discordgo.MessageEmbed{
		Title:       "⚔️ " + report.BossName,
		Description: description,
		Color:       0xf1c40f, // Золотой
		Footer: &discordgo.MessageEmbedFooter{
			Text: "LazyCatBot PVE Progression • Sirus.su",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	_, err := s.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		fmt.Println("Error sending embed:", err)
	}
}
