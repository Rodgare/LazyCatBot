package discord

import (
	"LazyCatBot/internal/storage"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type BotHandler struct {
	Store *storage.LeaderboardStorage
}

func (h *BotHandler) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.ToLower(m.Content) == "/топ" {

		response := "Таблица из стореджа"

		_, err := s.ChannelMessageSend(m.ChannelID, response)
		if err != nil {
			fmt.Println("Send message error:", err)
		}
	}
}
