package discord

import (
	"LazyCatBot/internal/models"
	"bytes"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

func formatKilledAt(s string) string {
	loc := time.FixedZone("MSK", 3*3600)
	t, err := time.ParseInLocation("2006-01-02 15:04:05", s, loc)
	if err != nil {
		return s
	}
	return t.Format("02-01-2006 15:04")
}

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

func SendKillReport(s *discordgo.Session, channelID string, report models.BossKillReport) error {
	embed := killReportEmbed(report)
	params := &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
	}
	if file := renderKillReportFile(report); file != nil {
		params.Files = []*discordgo.File{file}
	}

	_, err := s.ChannelMessageSendComplex(channelID, params)
	if err != nil {
		slog.Error("Error sending report", "error", err, "channel_id", channelID)
	}
	return err
}

// EditKillReport rewrites an existing message (the queue notice) into the
// finished report, attaching the rendered image to the same message.
func EditKillReport(s *discordgo.Session, channelID, messageID string, report models.BossKillReport) error {
	return editReportMessage(s, channelID, messageID, killReportEmbed(report), renderKillReportFile(report))
}

func killReportEmbed(report models.BossKillReport) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    report.GuildName,
			IconURL: fmt.Sprintf("https://sirus.su/api/base/%s/guild/%d/emblem.webp", report.Realm, report.GuildID),
			URL:     fmt.Sprintf("https://sirus.su/base/guilds/%s/%d", report.Realm, report.GuildID),
		},
		Title:  fmt.Sprintf("%s — %s", report.MapName, report.BossName),
		URL:    fmt.Sprintf("https://sirus.su/base/pve-progression/boss-kill/%s/%d", report.Realm, report.KillID),
		Color:  0xf1c40f,
		Fields: buildFields(report, report.Loots),
		Image: &discordgo.MessageEmbedImage{
			URL: "attachment://report.png",
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: generateJoke(),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func renderKillReportFile(report models.BossKillReport) *discordgo.File {
	imgData, err := RenderReportImage(report)
	if err != nil {
		slog.Error("Error rendering image", "error", err)
		return nil
	}
	return &discordgo.File{
		Name:        "report.png",
		ContentType: "image/png",
		Reader:      bytes.NewReader(imgData),
	}
}

// editReportMessage edits a message and adds an attachment. Discord requires
// new attachments to be listed in the attachments array with id equal to the
// file index, so this uses a raw multipart request instead of MessageEdit.
func editReportMessage(s *discordgo.Session, channelID, messageID string, embed *discordgo.MessageEmbed, file *discordgo.File) error {
	emptyContent := ""
	payload := struct {
		Content     *string                      `json:"content"`
		Embeds      []*discordgo.MessageEmbed    `json:"embeds"`
		Attachments []reportAttachment           `json:"attachments"`
		Components  []discordgo.MessageComponent `json:"components"`
	}{
		Content:     &emptyContent,
		Embeds:      []*discordgo.MessageEmbed{embed},
		Components:  []discordgo.MessageComponent{},
		Attachments: []reportAttachment{},
	}

	files := []*discordgo.File{}
	if file != nil {
		files = append(files, file)
		payload.Attachments = []reportAttachment{{ID: 0, Filename: file.Name}}
	} else {
		embed.Image = nil
	}

	contentType, body, err := discordgo.MultipartBodyWithJSON(payload, files)
	if err != nil {
		return err
	}
	_, err = s.RequestRaw("PATCH", discordgo.EndpointChannelMessage(channelID, messageID),
		contentType, body, discordgo.EndpointChannelMessage(channelID, ""), 0)
	return err
}

type reportAttachment struct {
	ID       int    `json:"id"`
	Filename string `json:"filename"`
}

func buildFields(report models.BossKillReport, lootsBlock []models.LootReport) []*discordgo.MessageEmbedField {
	var fields []*discordgo.MessageEmbedField

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Попытки",
		Value:  fmt.Sprintf("%d", report.Attempts),
		Inline: true,
	})

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Когда убили",
		Value:  formatKilledAt(report.KilledAt),
		Inline: true,
	})

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Время боя",
		Value:  fmt.Sprintf("%s", report.Duration),
		Inline: true,
	})

	// fields = append(fields, &discordgo.MessageEmbedField{
	// 	Name:   "\u200B",
	// 	Value:  "\u200B",
	// 	Inline: false,
	// })

	fields = append(fields, &discordgo.MessageEmbedField{
		Name: "Общий DPS",
		Value: fmt.Sprintf("```ansi\n%s%s%s\n```",
			ansiColors["red_bold"],
			FormatNum(report.TotalDps),
			ansiColors["reset"]),
		Inline: true,
	})

	fields = append(fields, &discordgo.MessageEmbedField{
		Name: "Общий HPS",
		Value: fmt.Sprintf("```ansi\n%s%s%s\n```",
			ansiColors["green_bold"],
			FormatNum(report.TotalHps),
			ansiColors["reset"]),
		Inline: true,
	})

	// fields = append(fields, &discordgo.MessageEmbedField{
	// 	Name:   "\u200B",
	// 	Value:  "\u200B",
	// 	Inline: false,
	// })

	if len(lootsBlock) > 0 {
		var lines []string
		totalItems := 0
		for _, item := range lootsBlock {
			if item.Name == "" {
				continue
			}
			totalItems++
			lines = append(lines, fmt.Sprintf("x%d [%s](https://sirus.su/base/item/%d/%s)\n",
				item.Count, item.Name, item.ID, report.Realm))
		}

		if totalItems > 0 {
			const limit = 1024
			var sb strings.Builder
			shown := 0
			for _, line := range lines {
				if sb.Len()+len(line) > limit {
					break
				}
				sb.WriteString(line)
				shown++
			}
			if remaining := totalItems - shown; remaining > 0 {
				note := fmt.Sprintf("…и ещё %d", remaining)
				if sb.Len()+len(note) <= limit {
					sb.WriteString(note)
				}
			}
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "Лут",
				Value:  sb.String(),
				Inline: false,
			})
		}
	}

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
		"LazyCatBot — Вы закрыли рейд? Отлично, освободите моё кресло. 🪑",
		"LazyCatBot — Опять весь рейд затащил один хил? Понятно. 💉",
		"LazyCatBot — Босс упал, но моральный ущерб мне никто не возместит. 🐈‍⬛",
		"LazyCatBot — Босс лежал, рейд стоял, кот спал. Все стабильно. 🐈",
	}

	return jokes[rand.Intn(len(jokes))]
}

type DiscordReporter struct {
	session *discordgo.Session

	mu        sync.RWMutex
	guildByCh map[string]string
}

func NewDiscordReporter(s *discordgo.Session) *DiscordReporter {
	return &DiscordReporter{
		session:   s,
		guildByCh: make(map[string]string),
	}
}

// RememberGuild records which server a channel belongs to. Premium is
// guild-wide, so the worker needs this mapping for channels it only knows by id.
func (r *DiscordReporter) RememberGuild(channelID, guildID string) {
	if channelID == "" || guildID == "" {
		return
	}
	r.mu.Lock()
	r.guildByCh[channelID] = guildID
	r.mu.Unlock()
}

// GetChannelGuildID returns the server id for a channel: from the cache first,
// then from the Discord API. Empty means unknown.
func (r *DiscordReporter) GetChannelGuildID(channelID string) string {
	if channelID == "" {
		return ""
	}
	r.mu.RLock()
	guildID := r.guildByCh[channelID]
	r.mu.RUnlock()
	if guildID != "" {
		return guildID
	}

	ch, err := r.session.Channel(channelID)
	if err != nil || ch == nil {
		return ""
	}
	r.RememberGuild(channelID, ch.GuildID)
	return ch.GuildID
}

func (r *DiscordReporter) SendKillReport(channelID string, report models.BossKillReport) error {
	if os.Getenv("DEBUG") == "true" {
		debugChan := os.Getenv("DEBUG_CHANNEL_ID")
		if debugChan != "" {
			fmt.Printf("[DEBUG MODE] Перенаправление отчёта с канала %s в DEBUG_CHANNEL_ID %s\n", channelID, debugChan)
			channelID = debugChan
		}
	}
	return SendKillReport(r.session, channelID, report)
}

func (r *DiscordReporter) EditKillReport(channelID, messageID string, report models.BossKillReport) error {
	return EditKillReport(r.session, channelID, messageID, report)
}

func (r *DiscordReporter) GetChannelName(channelID string) string {
	ch, err := r.session.Channel(channelID)
	if err != nil || ch == nil || ch.Name == "" {
		return channelID
	}
	return ch.Name
}

func (r *DiscordReporter) SendChannelNotice(channelID, text string) error {
	_, err := r.session.ChannelMessageSend(channelID, text)
	return err
}

// SendQueueNotice posts the "report is queued" placeholder for free channels
// and returns its message ID so the same message can be edited and removed.
func (r *DiscordReporter) SendQueueNotice(channelID, text string) (string, error) {
	msg, err := r.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content:    text,
		Components: queueNoticeComponents(),
	})
	if err != nil {
		return "", err
	}
	return msg.ID, nil
}

// EditQueueNotice updates an existing queue notice in place.
func (r *DiscordReporter) EditQueueNotice(channelID, messageID, text string) error {
	if messageID == "" {
		return nil
	}
	components := queueNoticeComponents()
	_, err := r.session.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:         messageID,
		Channel:    channelID,
		Content:    &text,
		Components: &components,
	})
	return err
}

func queueNoticeComponents() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "🐾 Премиум", Style: discordgo.SecondaryButton, CustomID: "premium_open"},
		}},
	}
}

func (r *DiscordReporter) DeleteNotice(channelID, messageID string) error {
	if messageID == "" {
		return nil
	}
	return r.session.ChannelMessageDelete(channelID, messageID)
}

// SendAnnouncement posts a rare one-way news message to the guild's
// system/welcome channel.
func (r *DiscordReporter) SendAnnouncement(channelID, text string) error {
	_, err := r.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: text,
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{Label: "🐾 Премиум", Style: discordgo.SecondaryButton, CustomID: "premium_open"},
			}},
		},
	})
	return err
}

// SendWeeklyDigest renders and posts the weekly premium image.
func (r *DiscordReporter) SendWeeklyDigest(channelID string, digest models.WeeklyDigest) error {
	img, err := RenderWeeklyDigest(digest)
	if err != nil {
		return err
	}

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("🐾 %s — итоги недели", digest.GuildName),
		Color: 0xf1c40f,
		Image: &discordgo.MessageEmbedImage{URL: "attachment://digest.png"},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "LazyCatBot · премиум-дайджест",
		},
	}

	params := &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Files: []*discordgo.File{
			{
				Name:        "digest.png",
				ContentType: "image/png",
				Reader:      bytes.NewReader(img),
			},
		},
	}

	_, err = r.session.ChannelMessageSendComplex(channelID, params)
	return err
}
