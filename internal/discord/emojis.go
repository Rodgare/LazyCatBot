package discord

import (
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var customEmojiRe = regexp.MustCompile(`<a?:\w+:\d+>`)

func stripCustomEmojis(s string) string {
	return customEmojiRe.ReplaceAllString(s, "")
}

func isEmojiAccessError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Missing Access") || strings.Contains(msg, "50001")
}

func sanitizeEmbed(embed *discordgo.MessageEmbed) *discordgo.MessageEmbed {
	cp := *embed
	cp.Title = stripCustomEmojis(cp.Title)
	cp.Description = stripCustomEmojis(cp.Description)

	if cp.Author != nil {
		a := *cp.Author
		a.Name = stripCustomEmojis(a.Name)
		cp.Author = &a
	}

	if cp.Footer != nil {
		f := *cp.Footer
		f.Text = stripCustomEmojis(f.Text)
		cp.Footer = &f
	}

	if len(cp.Fields) > 0 {
		fields := make([]*discordgo.MessageEmbedField, len(cp.Fields))
		for i, f := range cp.Fields {
			if f == nil {
				continue
			}
			fc := *f
			fc.Name = stripCustomEmojis(fc.Name)
			fc.Value = stripCustomEmojis(fc.Value)
			fields[i] = &fc
		}
		cp.Fields = fields
	}

	return &cp
}