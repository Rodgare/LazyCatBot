package discord

import (
	"bytes"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

func SendKillReport(s *discordgo.Session, channelID string, report BossKillReport) {
	// Генерируем изображение
	imgBytes, err := RenderReportImage(report)
	if err != nil {
		fmt.Println("Error rendering report image:", err)
		return
	}

	description := fmt.Sprintf("⏱️ **Duration:** %s  |  🔄 **Attempts:** %d\n💥 **Raid DPS:** %d",
		report.Duration, report.Attempts, report.TotalDps)

	embed := &discordgo.MessageEmbed{
		Title:       "⚔️Boss Kill: " + report.BossName,
		Description: description,
		Color:       0xf1c40f,
		Image: &discordgo.MessageEmbedImage{
			URL: "attachment://report.png",
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "LazyCatBot PVE Progression • Sirus.su",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	params := &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Files: []*discordgo.File{
			{
				Name:        "report.png",
				ContentType: "image/png",
				Reader:      bytes.NewReader(imgBytes),
			},
		},
	}

	_, err = s.ChannelMessageSendComplex(channelID, params)
	if err != nil {
		fmt.Println("Error sending report:", err)
	}
}
