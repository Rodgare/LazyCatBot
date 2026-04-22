package discord

import (
	"bytes"
	"fmt"
	"math/rand"
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
		Fields: buildFields(report, ddBlocks, healBlocks, report.Loots),
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

	imgData, err := RenderReportImage(report)
	if err == nil {
		params.Files = []*discordgo.File{
			{
				Name:        "report.png",
				ContentType: "image/png",
				Reader:      bytes.NewReader(imgData),
			},
		}
	} else {
		fmt.Println("Error rendering image:", err)
	}

	_, err = s.ChannelMessageSendComplex(channelID, params)
	if err != nil {
		fmt.Println("Error sending report:", err)
	}
}

func buildFields(report BossKillReport, ddBlocks, healBlocks []string, lootsBlock []LootReport) []*discordgo.MessageEmbedField {
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
		Inline: false,
	})

	for i, block := range ddBlocks {
		name := "ДД"
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
		Name:   "\u200B",
		Value:  "\u200B",
		Inline: false,
	})

	fields = append(fields, &discordgo.MessageEmbedField{
		Name: "Общий HPS",
		Value: fmt.Sprintf("```ansi\n%s%s%s\n```",
			ansiColors["green_bold"],
			FormatNum(report.TotalHps),
			ansiColors["reset"]),
		Inline: false,
	})

	for i, block := range healBlocks {
		name := "Хилы"
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
		Name:   "\u200B",
		Value:  "\u200B",
		Inline: false,
	})

	for i, block := range lootsBlock {
		name := "Лут"
		if i > 0 {
			name = "\u2800"
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   name,
			Value:  fmt.Sprintf("x%d [%s](https://sirus.su/base/item/%d/x3)", block.Count, block.Name, block.ID),
			Inline: false,
		})
	}

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "\u200B",
		Value:  "\u200B",
		Inline: false,
	})

	return fields
}

func generateJoke() string {
	jokes := []string{
		"LazyCatBot — Мяу, рейд пройден! 🐾",
		"LazyCatBot — Кот посмотрел логи и одобрил 🐈",
		"LazyCatBot — Мур-мур, прогресс засчитан 🐾",
		"LazyCatBot — Еще один рейд... Где мой вискас? 😒",
		"LazyCatBot — Сплю на клавиатуре, пока боссы падают 🐱💤",
		"LazyCatBot — Кот одобряет этот лут 💍",
		"LazyCatBot — Вы там босса били или спали? 🐾",
		"LazyCatBot — Этот рейд был легче, чем пакет с кормом 📦",
		"LazyCatBot — Мяу-аналитика завершена. Идите спать.",
		"LazyCatBot — Мог бы протащить всех, но мне лень 🐾",
		"LazyCatBot — 1% боссу остался? Зовите, когда будет 0% 🐈",
		"LazyCatBot — Опять мне логи разгребать... 🐈‍⬛",
		"LazyCatBot — Хватит вайпаться, мне спать пора. 🐱",
		"LazyCatBot — Ваш ДПС вызывает у меня экзистенциальный кризис. 😒",
		"LazyCatBot — Прогресс зафиксировал. Не благодарите. 🐾",
		"LazyCatBot — Опять вы за свое? Я только прилег. 💤",
		"LazyCatBot — Надеюсь, вы хоть не в луже стояли? 😒",
	}

	return jokes[rand.Intn(len(jokes))]
}
