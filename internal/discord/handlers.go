package discord

import (
	"LazyCatBot/internal/storage"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type BotHandler struct {
	LbStore  *storage.LeaderboardStorage
	SubStore *storage.SubscribeStorage
}

func (h *BotHandler) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, "/set") {
		parts := strings.Fields(m.Content)
		if len(parts) < 2 {
			s.ChannelMessageSend(m.ChannelID, "Для добавления гильдии в бот, используйте команду /set id_гильдии")
			return
		}
		guildId, err := strconv.Atoi(parts[1])
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "ID гильдии должен быть числом")
			return
		}

		err = h.SubStore.Subscribe(guildId, m.ChannelID, m.GuildID)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Ошибка сохранения подписки")
			return
		}

		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("✅ Теперь я слежу за гильдией %d в этом канале!", guildId))
	}

	if strings.ToLower(m.Content) == "/топ" {

		response := "Таблица из стореджа"

		_, err := s.ChannelMessageSend(m.ChannelID, response)
		if err != nil {
			fmt.Println("Send message error:", err)
		}
	}
}
