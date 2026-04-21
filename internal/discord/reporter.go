package discord

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

var ansiColors = map[string]string{
	// Базовые (яркие)
	"red":    "\033[31m",
	"green":  "\033[32m",
	"yellow": "\033[33m",
	"blue":   "\033[34m",
	"purple": "\033[35m",
	"cyan":   "\033[36m",
	"white":  "\033[37m",

	// Жирные (более насыщенные версии для акцентов)
	"red_bold":    "\033[1;31m",
	"green_bold":  "\033[1;32m",
	"yellow_bold": "\033[1;33m",
	"blue_bold":   "\033[1;34m",
	"purple_bold": "\033[1;35m",
	"cyan_bold":   "\033[1;36m",
	"white_bold":  "\033[1;37m",

	// Сброс цвета (обязательно в конце строки)
	"reset": "\033[0m",
}

func SendKillReport(s *discordgo.Session, channelID string, report BossKillReport) {
	ddBlocks, healBlocks := BuildReportText(report)

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    report.GuildName,
			IconURL: fmt.Sprintf("https://sirus.su/api/base/22/guild/%d/emblem.webp", report.GuildID),
			URL:     fmt.Sprintf("https://sirus.su/base/guilds/x3/%d", report.GuildID),
		},
		Title:  fmt.Sprintf("%s — %s", report.MapName, report.BossName),
		URL:    fmt.Sprintf("https://sirus.su/base/pve-progression/boss-kill/x3/%d", report.KillID),
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
		Name: "Общий DPS",
		Value: fmt.Sprintf("```ansi\n%s%s%s\n```",
			ansiColors["red_bold"],
			FormatNum(report.TotalDps),
			ansiColors["reset"]),
		Inline: true,
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
		Name: "Общий HPS",
		Value: fmt.Sprintf("```ansi\n%s%s%s\n```",
			ansiColors["green_bold"],
			FormatNum(report.TotalHps),
			ansiColors["reset"]),
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
