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
		"⏱️ **Duration:** %s  ·  🔄 **Attempts:** %d  ·  💥 **Raid DPS:** %s",
		report.Duration, report.Attempts, formatNum(report.TotalDps),
	)

	embed := &discordgo.MessageEmbed{
		Title:       "⚔️  Boss Kill: " + report.BossName,
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
		name := "⚔️  Damage Dealers"
		if i > 0 {
			name = "⚔️  Damage Dealers (cont.)"
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   name,
			Value:  block,
			Inline: false,
		})
	}

	for i, block := range healBlocks {
		name := "💚  Healers"
		if i > 0 {
			name = "💚  Healers (cont.)"
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   name,
			Value:  block,
			Inline: false,
		})
	}

	return fields
}
