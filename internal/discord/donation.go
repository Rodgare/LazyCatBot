package discord

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Donation holds the public support settings for the bot. It is rendered as
// always-available link buttons on reports and as a link block in /help and the
// guild welcome embed. Everything is hidden when no URL is set, so the feature
// is a no-op until configured.
type Donation struct {
	URL       string
	CryptoURL string
	Title     string
}

func (d Donation) Enabled() bool {
	return d.URL != "" || d.CryptoURL != ""
}

func (d Donation) effectiveTitle() string {
	if d.Title != "" {
		return d.Title
	}
	return "Поддержать проект"
}

// Components returns the link buttons for reports (card + optional crypto).
func (d Donation) Components() []discordgo.MessageComponent {
	if !d.Enabled() {
		return nil
	}
	var buttons []discordgo.MessageComponent
	if d.URL != "" {
		buttons = append(buttons, discordgo.Button{
			Label: "🐾 Закинуть на корм",
			Style: discordgo.LinkButton,
			URL:   d.URL,
		})
	}
	if d.CryptoURL != "" {
		buttons = append(buttons, discordgo.Button{
			Label: "💎 Крипта",
			Style: discordgo.LinkButton,
			URL:   d.CryptoURL,
		})
	}
	if len(buttons) == 0 {
		return nil
	}
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: buttons}}
}

// Summary returns a markdown block for /help and the welcome embed.
func (d Donation) Summary() string {
	if !d.Enabled() {
		return ""
	}
	var b strings.Builder
	b.WriteString("**🐾 " + d.effectiveTitle() + ":**\n")
	b.WriteString("В сообщении доната укажи свой игровой ник — он будет подсвечен в отчётах. Спасибо за поддержку!")
	if d.URL != "" {
		b.WriteString("\n[🐾 Оплатить корм](" + d.URL + ")")
	}
	if d.CryptoURL != "" {
		b.WriteString(" · [💎 Крипта](" + d.CryptoURL + ")")
	}
	return b.String()
}

// SummaryBlock returns Summary prefixed with blank lines for appending to an
// embed description, or an empty string when the feature is disabled.
func (d Donation) SummaryBlock() string {
	s := d.Summary()
	if s == "" {
		return ""
	}
	return "\n\n" + s
}
