package discord

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

func SendKillReport(s *discordgo.Session, channelID string, report BossKillReport) {
	ddBlock, healBlock := BuildReportText(report)

	// Основное описание
	description := fmt.Sprintf(
		"⏱️ **Duration:** %s  ·  🔄 **Attempts:** %d  ·  💥 **Raid DPS:** %s",
		report.Duration, report.Attempts, formatNum(report.TotalDps),
	)

	embed := &discordgo.MessageEmbed{
		Title:       "⚔️  Boss Kill: " + report.BossName,
		Description: description,
		Color:       0xf1c40f,
		Fields:      buildFields(ddBlock, healBlock),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "LazyCatBot PVE Progression • Sirus.su",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	params := &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
	}

	_, err := s.ChannelMessageSendComplex(channelID, params)
	if err != nil {
		fmt.Println("Error sending report:", err)
	}
}

func buildFields(ddBlock, healBlock string) []*discordgo.MessageEmbedField {
	var fields []*discordgo.MessageEmbedField

	if ddBlock != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "⚔️  Damage Dealers",
			Value:  ddBlock,
			Inline: false,
		})
	}

	if healBlock != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "💚  Healers",
			Value:  healBlock,
			Inline: false,
		})
	}

	return fields
}
