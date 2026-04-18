package discord

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

func SendKillReport(s *discordgo.Session, channelID string, report BossKillReport) {
	var playerList string

	for _, p := range report.Players {
		playerList += fmt.Sprintf("**%s** — DPS: `%d` — Rank: **#%d**\n", p.Name, p.Dps, p.SpecRank)
	}

	embed := &discordgo.MessageEmbed{
		Title:       report.BossName,
		Description: playerList,
		Color:       0x00ff00,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	s.ChannelMessageSendEmbed(channelID, embed)
}
