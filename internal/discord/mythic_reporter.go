package discord

import (
	"LazyCatBot/internal/models"
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (r *DiscordReporter) SendMythicReport(channelID string, report *models.MythicReport) {
	embed := &discordgo.MessageEmbed{
		Title:  fmt.Sprintf("%s — %d +%d", report.Name, report.KeystoneLevel, report.RewardLevel),
		URL:    fmt.Sprintf("https://sirus.su/base/ladder/keystone/run/%s/%d", report.Realm, report.ID),
		Color:  0xf1c40f,
		Fields: r.buildMythicFields(report),
		Image: &discordgo.MessageEmbedImage{
			URL: "attachment://report.png",
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: generateJoke(),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	params := &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
	}

	imgData, err := r.RenderMythicReportImage(report)
	if err == nil {
		params.Files = []*discordgo.File{
			{
				Name:        "report.png",
				ContentType: "image/png",
				Reader:      bytes.NewReader(imgData),
			},
		}
	} else {
		fmt.Println("Error mythic rendering image:", err)
	}

	_, err = r.session.ChannelMessageSendComplex(channelID, params)
	if err != nil {
		fmt.Println("Error sending mythic report:", err)
	}
}

func (r *DiscordReporter) buildMythicFields(report *models.MythicReport) []*discordgo.MessageEmbedField {
	var fields []*discordgo.MessageEmbedField

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Дата",
		Value:  fmt.Sprintf("<t:%d:d> <t:%d:t>", report.DateTime, report.DateTime),
		Inline: true,
	})

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Таймер",
		Value:  fmt.Sprintf("%d", report.DurationMs),
		Inline: true,
	})

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Очки",
		Value:  fmt.Sprintf("%.3f", report.Score),
		Inline: true,
	})

	var affixes []string
	for _, affixID := range report.Affixes {
		if affix, ok := models.AffixMap[affixID]; ok {
			affixes = append(affixes, fmt.Sprintf("%s %s", affix.Emoji, affix.Name))
		}
	}

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Аффиксы",
		Value:  strings.Join(affixes, "\n"),
		Inline: false,
	})

	return fields
}
