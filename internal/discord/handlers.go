package discord

import (
	"LazyCatBot/internal/storage"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type BotHandler struct {
	LbStore  *storage.LeaderboardStorage
	SubStore *storage.SubscribeStorage
}

func (h *BotHandler) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
}

func (h *BotHandler) InteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := i.ApplicationCommandData()
	guildID := i.GuildID
	channelID := i.ChannelID

	switch data.Name {
	case "set":
		guildNum := int(data.Options[0].IntValue())
		err := h.SubStore.Subscribe(guildNum, channelID, guildID)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Ошибка при сохранении подписки.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ Теперь я слежу за гильдией %d в этом канале!", guildNum),
			},
		})

	case "unset":
		guildNum := int(data.Options[0].IntValue())
		err := h.SubStore.Unsubscribe(guildNum, channelID, guildID)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Ошибка при удалении подписки.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Я больше не слежу за гильдией %d в этом канале.", guildNum),
			},
		})

	case "help":
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{
					{
						Title: "🐈 Справка LazyCatBot",
						Description: "Я помогаю отслеживать прогресс гильдий на Sirus.su!\n\n" +
							"**/set [id]** — Подписаться на отчеты гильдии в этом канале.\n" +
							"**/unset [id]** — Отписаться от отчетов.\n" +
							"**/help** — Показать это сообщение.",
						Color: 0xf1c40f,
					},
				},
			},
		})
	}
}

func (h *BotHandler) GuildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	var channelID string
	if g.SystemChannelID != "" {
		channelID = g.SystemChannelID
	} else {
		for _, ch := range g.Channels {
			if ch.Type == discordgo.ChannelTypeGuildText {
				channelID = ch.ID
				break
			}
		}
	}

	if channelID == "" {
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "🐈 Привет! Я LazyCatBot",
		Description: "Я помогу вам отслеживать прогресс убийства боссов вашей гильдии на Sirus.su!\n\n" +
			"**Как меня настроить:**\n" +
			"Просто введите `/` и выберите команду **/set** из списка.\n\n" +
			"*(ID гильдии можно найти в ссылке на вашу гильдию в базе Sirus)*",
		Color: 0xf1c40f,
	}

	s.ChannelMessageSendEmbed(channelID, embed)
}
