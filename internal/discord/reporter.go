package discord

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

func SendKillReport(s *discordgo.Session, channelID string, report BossKillReport) {
	ddBlocks, healBlocks := BuildReportText(report)

	embed := &discordgo.MessageEmbed{
		Title:  "⚔️  Boss: " + report.BossName,
		Color:  0xf1c40f,
		Fields: buildFields(report, ddBlocks, healBlocks),
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

func buildFields(report BossKillReport, ddBlocks []string, healBlocks []string) []*discordgo.MessageEmbedField {
	var fields []*discordgo.MessageEmbedField

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Попытки",
		Value:  fmt.Sprintf("%d", report.Attempts),
		Inline: true,
	})

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Когда убили",
		Value:  fmt.Sprintf("%s", report.KilledAt),
		Inline: true,
	})

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Время боя",
		Value:  fmt.Sprintf("%s", report.Duration),
		Inline: true,
	})

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "\u200B",
		Value:  "\u200B",
		Inline: false,
	})

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Общий DPS",
		Value:  fmt.Sprintf("```diff\n-%s\n```", FormatNum(report.TotalDps)),
		Inline: false,
	})

	for i, block := range ddBlocks {
		name := "#\u2800Ник\u2800Дпс\u2800(Рейтинг по спеку)"
		if i > 0 {
			name = "\u2800"
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   name,
			Value:  block,
			Inline: false,
		})
	}

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Общий HPS",
		Value:  fmt.Sprintf("```diff\n+%s\n```", FormatNum(report.TotalHps)),
		Inline: true,
	})

	for i, block := range healBlocks {
		name := "#\u2800Ник\u2800Хпс\u2800(Рейтинг по спеку)"
		if i > 0 {
			name = "\u2800"
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   name,
			Value:  block,
			Inline: false,
		})
	}

	return fields
}
