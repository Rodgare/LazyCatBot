package discord

import (
	"LazyCatBot/internal/sirus"
	"LazyCatBot/internal/storage"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

type BotHandler struct {
	LbStore        *storage.LeaderboardStorage
	SubStore       *storage.SubscribeStorage
	PlayerSubStore *storage.PlayerSubscribeStorage
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

	case "setcat":
		name := data.Options[0].StringValue()
		id, err := sirus.FetchPlayerID(name)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Не удалось найти персонажа «%s». Проверьте правильность написания.", name),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		err = h.PlayerSubStore.Subscribe(id, name, channelID, guildID)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Ошибка при сохранении подписки в базу данных.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ Теперь я слежу за игроком **%s** (ID: %d) в этом канале!", name, id),
			},
		})

	case "unsetcat":
		playerID := int(data.Options[0].IntValue())
		err := h.PlayerSubStore.Unsubscribe(playerID, channelID, guildID)
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
				Content: fmt.Sprintf("❌ Я больше не слежу за игроком %d в этом канале.", playerID),
			},
		})

	case "list":
		guilds, err := h.SubStore.GetGuildsByChannel(channelID)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Ошибка при получении списка гильдий.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		if len(guilds) == 0 {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "В этом канале не отслеживается ни одна гильдия.",
				},
			})
			return
		}

		content := "📊 **Отслеживаемые гильдии в этом канале:**\n"
		for _, id := range guilds {
			content += fmt.Sprintf("— Гильдия ID `%d`\n", id)
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
			},
		})

	case "listcats":
		characters, err := h.PlayerSubStore.GetCharactersByChannel(channelID)
		if err != nil {
			log.Printf("listcats err: %v", err)
			return
		}
		if len(characters) == 0 {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "В этом канале не отслеживается ни один игрок.",
				},
			})
			return
		}
		content := "📊 **Отслеживаемые игроки в этом канале:**\n"
		for _, id := range characters {
			content += fmt.Sprintf("— Игрок ID `%d`\n", id)
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
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
							"**/set id_гильдии** — Подписаться на отчеты гильдии в этом канале.\n" +
							"**/setcat id_игрока** - Подписаться на отчеты конкретного игрока\n" +
							"**/unset id_гильдии** — Отписаться от отчетов гильдии.\n" +
							"**/unsetcat id_игрока** — Отписаться от отчетов.\n" +
							"**/list** — Список отслеживаемых гильдий в данном канале.\n" +
							"**/listcats** — Список отслеживаемых игроков в данном канале.\n" +
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
		Description: "Я помогу вам отслеживать убийства боссов вашей гильдии!\n\n" +
			"**Как меня настроить:**\n" +
			"Создайте текстовый канал и введите в этом канале комманду /set id_гильдии (/set 1234)\n\n" +
			"Список комманд /help\n\n" +
			"*(ID гильдии можно найти в ссылке на вашу гильдию на сайте Sirus)*",
		Color: 0xf1c40f,
	}

	s.ChannelMessageSendEmbed(channelID, embed)
}
