package discord

import (
	"LazyCatBot/internal/sirus"
	"LazyCatBot/internal/storage"
	"fmt"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type BotHandler struct {
	sirusClient    *sirus.Client
	LbStore        *storage.LeaderboardStorage
	SubStore       *storage.SubscribeStorage
	PlayerSubStore *storage.PlayerSubscribeStorage
	GMStore        *storage.GuildMembersStorage
	Logger         *slog.Logger
}

func NewHandler(
	sc *sirus.Client,
	lb *storage.LeaderboardStorage,
	sub *storage.SubscribeStorage,
	pSub *storage.PlayerSubscribeStorage,
	gm *storage.GuildMembersStorage,
	logger *slog.Logger,
) *BotHandler {
	return &BotHandler{
		sirusClient:    sc,
		LbStore:        lb,
		SubStore:       sub,
		PlayerSubStore: pSub,
		GMStore:        gm,
		Logger:         logger,
	}
}

func (h *BotHandler) InteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		h.HandleSlashCommands(s, i)
	case discordgo.InteractionMessageComponent:
		h.HandleButtons(s, i)
	case discordgo.InteractionModalSubmit:
		h.HandleModalSubmit(s, i)
	}
}

func (h *BotHandler) HandleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()

	l := h.Logger.With(
		"command_modal", data.CustomID,
		"guild_id", i.GuildID,
		"user", i.Member.User.Username,
	)

	switch data.CustomID {
	case "modal_add_player":
		playerName := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		id, _ := h.sirusClient.FetchPlayerID(playerName)
		err := h.PlayerSubStore.Subscribe(id, playerName, i.ChannelID, i.GuildID)

		content := fmt.Sprintf("✅ Игрок **%s** успешно добавлен в список отслеживания!", playerName)
		if err != nil {
			content = "❌ Ошибка при добавлении игрока."
			l.Error("Add player in modal window error", "error", err)
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	case "modal_add_guild":
		guildIDStr := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		var guildID int
		fmt.Sscanf(guildIDStr, "%d", &guildID)
		err := h.SubStore.Subscribe(guildID, i.ChannelID, i.GuildID)
		content := fmt.Sprintf("✅ Гильдия **%d** успешно добавлена!", guildID)
		if err != nil {
			content = "❌ Ошибка при добавлении гильдии."
			l.Error("Add guild in modal window error", "error", err)
		}

		go func(id int) {
			members, err := h.sirusClient.FetchGuildMembers(id)
			if err == nil && members != nil {
				h.GMStore.UpdateGuildMembers(id, *members)
				l.Info("Guild members are saved")
			}
		}(guildID)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})

	}

}

func (h *BotHandler) HandleButtons(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.MessageComponentData()

	if after, ok := strings.CutPrefix(data.CustomID, "toggle_reports_"); ok {
		var gID int
		fmt.Sscanf(after, "%d", &gID)
		newState, _ := h.SubStore.ToggleReports(gID, i.ChannelID)
		content := "🔔 Отчеты включены"
		if !newState {
			content = "🔕 Отчеты выключены"
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if after, ok := strings.CutPrefix(data.CustomID, "remove_player_"); ok {
		playerIDStr := after
		var pID int
		fmt.Sscanf(playerIDStr, "%d", &pID)

		h.PlayerSubStore.Unsubscribe(pID, i.ChannelID, i.GuildID)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "✅ Игрок удален из отслеживания.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if after, ok := strings.CutPrefix(data.CustomID, "remove_guild_"); ok {
		guildIDStr := after
		var gID int
		fmt.Sscanf(guildIDStr, "%d", &gID)

		h.SubStore.Unsubscribe(gID, i.ChannelID, i.GuildID)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ Гильдия %d удалена из отслеживания.", gID),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if after, ok := strings.CutPrefix(data.CustomID, "top_dps_"); ok {
		var raidID, bossID int
		fmt.Sscanf(after, "%d_%d", &raidID, &bossID)
		h.SendBossRanking(s, i, raidID, bossID, "dps")
		return
	}

	if after, ok := strings.CutPrefix(data.CustomID, "top_hps_"); ok {
		var raidID, bossID int
		fmt.Sscanf(after, "%d_%d", &raidID, &bossID)
		h.SendBossRanking(s, i, raidID, bossID, "hps")
		return
	}

	switch data.CustomID {
	case "btn_add_player":
		h.handleAddPlayerModal(s, i)
	case "btn_add_guild":
		h.handleAddGuildModal(s, i)
	}
}

func (h *BotHandler) handleAddGuildModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "modal_add_guild",
			Title:    "Добавление гильдии",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "guild_id",
							Label:       "ID гильдии на Сирусе",
							Style:       discordgo.TextInputShort,
							Placeholder: "Например: 12345",
							Required:    true,
						},
					},
				},
			},
		},
	})
}

func (h *BotHandler) handleAddPlayerModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "modal_add_player",
			Title:    "Добавление игрока для отслеживания",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "player_name",
							Label:       "Никнейм персонажа",
							Style:       discordgo.TextInputShort,
							Placeholder: "Например: Васяпро",
							Required:    true,
						},
					},
				},
			},
		},
	})
	if err != nil {
		h.Logger.Error("sending modal error", "error", err)
	}
}

func (h *BotHandler) HandleMenuCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guilds, _ := h.SubStore.GetGuildsByChannel(i.ChannelID)
	players, _ := h.PlayerSubStore.GetPlayersByChannel(i.ChannelID)

	var rows []discordgo.MessageComponent

	rows = append(rows, discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "🏰 + Гильдию", Style: discordgo.PrimaryButton, CustomID: "btn_add_guild"},
			discordgo.Button{Label: "👤 + Игрока", Style: discordgo.PrimaryButton, CustomID: "btn_add_player"},
		},
	})

	if len(guilds) > 0 {
		for _, gID := range guilds {
			enabled := h.SubStore.IsReportsEnabled(gID, i.ChannelID)

			guildRow := discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    fmt.Sprintf("🗑️ Гильдия ID: %d", gID),
						Style:    discordgo.DangerButton,
						CustomID: fmt.Sprintf("remove_guild_%d", gID),
					},
					discordgo.Button{
						Label:    map[bool]string{true: "🔔 Отчеты: ВКЛЮЧЕНЫ", false: "🔕 Отчеты: ВЫКЛЮЧЕНЫ"}[enabled],
						Style:    map[bool]discordgo.ButtonStyle{true: discordgo.SuccessButton, false: discordgo.SecondaryButton}[enabled],
						CustomID: fmt.Sprintf("toggle_reports_%d", gID),
					},
				},
			}
			rows = append(rows, guildRow)

			if len(rows) >= 5 {
				break
			}
		}
	}

	if len(players) > 0 {
		var playerButtons []discordgo.MessageComponent
		for pID, pName := range players {
			playerButtons = append(playerButtons, discordgo.Button{
				Label:    fmt.Sprintf("✖️ %s", pName),
				Style:    discordgo.DangerButton,
				CustomID: fmt.Sprintf("remove_player_%d", pID),
			})
			if len(playerButtons) == 5 {
				rows = append(rows, discordgo.ActionsRow{Components: playerButtons})
				playerButtons = []discordgo.MessageComponent{}
			}
			if len(rows) >= 5 {
				break
			}
		}
		if len(playerButtons) > 0 && len(rows) < 5 {
			rows = append(rows, discordgo.ActionsRow{Components: playerButtons})
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    "⚙️ **Управление трекингом в этом канале**",
			Components: rows,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

func (h *BotHandler) HandleTopMCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	raids := sirus.GetCheckBosses(22)

	var dpsButtons []discordgo.MessageComponent
	var hpsButtons []discordgo.MessageComponent

	for raidID, bosses := range raids {
		for _, boss := range bosses {
			dpsButtons = append(dpsButtons, discordgo.Button{
				Label:    boss.Name,
				Style:    discordgo.PrimaryButton,
				CustomID: fmt.Sprintf("top_dps_%d_%d", raidID, boss.ID),
				Emoji:    &discordgo.ComponentEmoji{Name: "⚔️"},
			})

			hpsButtons = append(hpsButtons, discordgo.Button{
				Label:    boss.Name,
				Style:    discordgo.SuccessButton,
				CustomID: fmt.Sprintf("top_hps_%d_%d", raidID, boss.ID),
				Emoji:    &discordgo.ComponentEmoji{Name: "🌿"},
			})
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "🏆 **Рейтинг за 2 кд:**",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: dpsButtons},
				discordgo.ActionsRow{Components: hpsButtons},
			},
		},
	})
}

func (h *BotHandler) HandleSlashCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()

	l := h.Logger.With(
		"command", data.Name,
		"guild_id", i.GuildID,
		"user", i.Member.User.Username,
	)

	l.Info("slash command received")

	guildID := i.GuildID
	channelID := i.ChannelID
	switch data.Name {
	case "menu":
		h.HandleMenuCommand(s, i)
	case "set":
		guildId := int(data.Options[0].IntValue())
		err := h.SubStore.Subscribe(guildId, channelID, guildID)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Ошибка при сохранении подписки.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			l.Error("Subscription saving error", "error", err)
			return
		}

		go func(id int) {
			members, err := h.sirusClient.FetchGuildMembers(id)
			if err == nil && members != nil {
				h.GMStore.UpdateGuildMembers(id, *members)
				l.Info("fresh guild members loaded", "sirus_guild_id", id)
			}
		}(guildId)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ Теперь я слежу за гильдией %d в этом канале!", guildId),
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
			l.Error("Subscription deleting error", "error", err)
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
		id, err := h.sirusClient.FetchPlayerID(name)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Не удалось найти персонажа «%s». Проверьте правильность написания.", name),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			l.Error("Search character error", "error", err)
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
			l.Error("Saving subscription to db error", "error", err)
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
			l.Error("Deleting subsription error", "error", err)
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
			l.Error("Getting guilds list error", "error", err)
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

		var content strings.Builder
		content.WriteString("📊 **Отслеживаемые гильдии в этом канале:**\n")
		for _, id := range guilds {
			fmt.Fprintf(&content, "— Гильдия ID `%d`\n", id)
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content.String(),
			},
		})
	case "topm":
		h.HandleTopMCommand(s, i)

	case "listcats":
		characters, err := h.PlayerSubStore.GetPlayersByChannel(channelID)
		if err != nil {
			l.Error("listcats error", "error", err)
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
		var content strings.Builder
		content.WriteString("📊 **Отслеживаемые игроки в этом канале:**\n")
		for id, name := range characters {
			fmt.Fprintf(&content, "— Игрок ID `%d`, Имя `%s`\n", id, name)
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content.String(),
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
							"**/setcat имя_игрока** - Подписаться на отчеты конкретного игрока\n" +
							"**/unset id_гильдии** — Отписаться от отчетов гильдии.\n" +
							"**/unsetcat имя_игрока** — Отписаться от отчетов игрока.\n" +
							"**/menu Вызывает меню с настройкой подписок бота, трекинга и т.д.\n" +
							"**/topm Команда вызывает меню с кнопками, который видят все, нажав на которые можно отправить рейтинги по чек босам среди игроков гильдии\n" +
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
	if h.SubStore.IsDiscordGuildSubscribed(g.ID) {
		h.Logger.Info("guild already subscribed", "discord_guild_id", g.ID)
		return
	}

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
		h.Logger.Info("empty channel ID")
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
