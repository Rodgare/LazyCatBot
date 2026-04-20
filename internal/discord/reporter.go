package discord

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

func SendKillReport(s *discordgo.Session, channelID string, report BossKillReport) {
	ddBlocks, healBlocks := BuildReportText(report)

	// Основное описание
	description := fmt.Sprintf(
		"⏱️ **Длительность боя:** %s  ·  🔄 **Попытки:** %d  ·  💥 **Raid DPS:** %s",
		report.Duration, report.Attempts, formatNum(report.TotalDps),
	)

	embed := &discordgo.MessageEmbed{
		Title:       "⚔️  Boss: " + report.BossName,
		Description: description,
		Color:       0xf1c40f,
		Fields:      buildFields(ddBlocks, healBlocks),
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

func buildFields(ddBlocks []string, healBlocks []string) []*discordgo.MessageEmbedField {
	var fields []*discordgo.MessageEmbedField

	for i, block := range ddBlocks {
		name := "⚔️  ДД"
		if i > 0 {
			name = "\u200b"
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   name,
			Value:  block,
			Inline: false,
		})
	}

	for i, block := range healBlocks {
		name := "💚  Хилы"
		if i > 0 {
			name = "\u200b"
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   name,
			Value:  block,
			Inline: false,
		})
	}

	return fields
}
