package discord

import (
	"LazyCatBot/internal/sirus"
	"LazyCatBot/internal/storage"
	"bytes"
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
	arStore        *storage.ActualRaidsStorage
	MythicStore    *storage.MythicLeaderboardStorage
	Logger         *slog.Logger
}

func NewHandler(
	sc *sirus.Client,
	lb *storage.LeaderboardStorage,
	sub *storage.SubscribeStorage,
	pSub *storage.PlayerSubscribeStorage,
	gm *storage.GuildMembersStorage,
	ar *storage.ActualRaidsStorage,
	mythic *storage.MythicLeaderboardStorage,
	logger *slog.Logger,
) *BotHandler {
	return &BotHandler{
		sirusClient:    sc,
		LbStore:        lb,
		SubStore:       sub,
		PlayerSubStore: pSub,
		GMStore:        gm,
		arStore:        ar,
		MythicStore:    mythic,
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

	if after, ok := strings.CutPrefix(data.CustomID, "modal_add_player_"); ok {
		realm := after
		playerName := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		playerName = strings.TrimSpace(playerName)

		id, err := h.sirusClient.FetchPlayerID(realm, playerName)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("❌ Не удалось найти персонажа «%s» на сервере %s. Проверьте имя.", playerName, strings.ToUpper(realm)),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			l.Error("Search character in modal error", "error", err)
			return
		}

		err = h.PlayerSubStore.Subscribe(id, playerName, i.ChannelID, i.GuildID, realm)
		content := fmt.Sprintf("✅ Игрок **%s** (ID: %d) [%s] успешно добавлен!", playerName, id, strings.ToUpper(realm))
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
		return
	}

	if after, ok := strings.CutPrefix(data.CustomID, "modal_add_guild_"); ok {
		realm := after
		guildIDStr := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
		var guildID int
		fmt.Sscanf(guildIDStr, "%d", &guildID)

		err := h.SubStore.Subscribe(guildID, i.ChannelID, i.GuildID, realm)
		content := fmt.Sprintf("✅ Гильдия **%d** [%s] успешно добавлена!", guildID, strings.ToUpper(realm))
		if err != nil {
			content = "❌ Ошибка при добавлении гильдии."
			l.Error("Add guild in modal window error", "error", err)
		}

		go func(id int, r string) {
			members, guildName, err := h.sirusClient.FetchGuildMembers(r, id)
			if err == nil && members != nil {
				h.GMStore.UpdateGuildMembers(r, id, *members)
				l.Info("Guild members are saved")
				if guildName != "" {
					h.SubStore.SetGuildName(id, r, guildName)
				}
			}
		}(guildID, realm)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

}

func (h *BotHandler) HandleButtons(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.MessageComponentData()

	if strings.HasPrefix(data.CustomID, "select_raid_") {
		if len(data.Values) > 0 {
			selectedValue := data.Values[0]
			if after, ok := strings.CutPrefix(selectedValue, "top_dps_"); ok {
				var raidID, bossID int
				fmt.Sscanf(after, "%d_%d", &raidID, &bossID)
				h.SendBossRanking(s, i, raidID, bossID, "dps")
				return
			}
			if after, ok := strings.CutPrefix(selectedValue, "top_hps_"); ok {
				var raidID, bossID int
				fmt.Sscanf(after, "%d_%d", &raidID, &bossID)
				h.SendBossRanking(s, i, raidID, bossID, "hps")
				return
			}
		}
		return
	}

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

	if data.CustomID == "toggle_mythic_reports" {
		newState, _ := h.SubStore.ToggleMythicReports(i.ChannelID)
		content := "🔔 Мифик-отчёты включены"
		if !newState {
			content = "🔕 Мифик-отчёты выключены"
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

	if after, ok := strings.CutPrefix(data.CustomID, "realm_select_guild_"); ok {
		h.handleAddGuildModalWithRealm(s, i, after)
		return
	}

	if after, ok := strings.CutPrefix(data.CustomID, "realm_select_player_"); ok {
		h.handleAddPlayerModalWithRealm(s, i, after)
		return
	}

	switch data.CustomID {
	case "btn_add_player":
		h.handlePromptSelectRealm(s, i, "player")
	case "btn_add_guild":
		h.handlePromptSelectRealm(s, i, "guild")
	}
}

func (h *BotHandler) handlePromptSelectRealm(s *discordgo.Session, i *discordgo.InteractionCreate, targetType string) {
	prefix := fmt.Sprintf("realm_select_%s_", targetType)
	targetName := map[string]string{"guild": "гильдии", "player": "игрока"}[targetType]

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("🌐 **Выберите игровой сервер для добавления %s:**", targetName),
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{Label: "🔮 Neverest x3", Style: discordgo.PrimaryButton, CustomID: prefix + "x3"},
						discordgo.Button{Label: "⚡ Sirus x5", Style: discordgo.PrimaryButton, CustomID: prefix + "x5"},
					},
				},
			},
		},
	})
}

func (h *BotHandler) handleAddGuildModalWithRealm(s *discordgo.Session, i *discordgo.InteractionCreate, realm string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("modal_add_guild_%s", realm),
			Title:    fmt.Sprintf("Добавление гильдии (%s)", strings.ToUpper(realm)),
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

func (h *BotHandler) handleAddPlayerModalWithRealm(s *discordgo.Session, i *discordgo.InteractionCreate, realm string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("modal_add_player_%s", realm),
			Title:    fmt.Sprintf("Добавление игрока (%s)", strings.ToUpper(realm)),
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

	mythicEnabled := h.SubStore.IsMythicReportsEnabled(i.ChannelID)

	rows = append(rows, discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "🏰 Добавить гильдию", Style: discordgo.PrimaryButton, CustomID: "btn_add_guild"},
			discordgo.Button{Label: "👤 Добавить игрока", Style: discordgo.PrimaryButton, CustomID: "btn_add_player"},
			discordgo.Button{
				Label:    map[bool]string{true: "🔔 Мифик: ВКЛ", false: "🔕 Мифик: ВЫКЛ"}[mythicEnabled],
				Style:    map[bool]discordgo.ButtonStyle{true: discordgo.SuccessButton, false: discordgo.SecondaryButton}[mythicEnabled],
				CustomID: "toggle_mythic_reports",
			},
		},
	})

	if len(guilds) > 0 {
		for _, gID := range guilds {
			enabled := h.SubStore.IsReportsEnabled(gID, i.ChannelID)

			guildRow := discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    fmt.Sprintf("Удалить Гильдия ID: %d", gID),
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
				Label:    fmt.Sprintf("✖️Удалить %s", pName),
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
	subs, err := h.SubStore.GetGuildSubsByChannel(i.ChannelID)
	if err != nil || len(subs) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ В этом канале не настроено отслеживание гильдий.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	realm := subs[0].Realm
	actualRaids, err := h.arStore.GetActualRaids(realm)
	if err != nil {
		h.Logger.Error("Error getting actual raids", "error", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Ошибка при получении актуальных рейдов из базы данных.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if len(actualRaids) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "📭 Список актуальных рейдов пуст",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	var raidIDs []int
	bossesByRaid := make(map[int][]int)
	seenRaids := make(map[int]bool)

	for _, ar := range actualRaids {
		if !seenRaids[ar.RaidID] {
			seenRaids[ar.RaidID] = true
			raidIDs = append(raidIDs, ar.RaidID)
		}
		bossesByRaid[ar.RaidID] = append(bossesByRaid[ar.RaidID], ar.BossID)
	}

	var rows []discordgo.MessageComponent

	for _, raidID := range raidIDs {
		raidName := sirus.GetRaidNameByID(raidID)
		if raidName == "" {
			raidName = fmt.Sprintf("Рейд %d", raidID)
		}

		var options []discordgo.SelectMenuOption
		bossIDs := bossesByRaid[raidID]

		for _, bossID := range bossIDs {
			bossName := sirus.GetBossName(raidID, bossID)
			if bossName == "" {
				bossName = fmt.Sprintf("Босс ID %d", bossID)
			}

			options = append(options, discordgo.SelectMenuOption{
				Label:       fmt.Sprintf("%s (⚔️ DPS)", bossName),
				Value:       fmt.Sprintf("top_dps_%d_%d", raidID, bossID),
				Description: fmt.Sprintf("Топ DPS на боссе %s", bossName),
				Emoji: &discordgo.ComponentEmoji{
					Name: "⚔️",
				},
			})

			options = append(options, discordgo.SelectMenuOption{
				Label:       fmt.Sprintf("%s (🌿 HPS)", bossName),
				Value:       fmt.Sprintf("top_hps_%d_%d", raidID, bossID),
				Description: fmt.Sprintf("Топ HPS на боссе %s", bossName),
				Emoji: &discordgo.ComponentEmoji{
					Name: "🌿",
				},
			})
		}

		if len(options) > 25 {
			options = options[:25]
		}

		if len(options) == 0 {
			continue
		}

		menu := discordgo.SelectMenu{
			CustomID:    fmt.Sprintf("select_raid_%d", raidID),
			Placeholder: fmt.Sprintf("🏰 %s", raidName),
			MenuType:    discordgo.StringSelectMenu,
			Options:     options,
		}
		rows = append(rows, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{menu},
		})

		if len(rows) >= 5 {
			break
		}
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    "🏆 **Рейтинг за 2 кд:**",
			Components: rows,
		},
	})
}

func (h *BotHandler) HandleTopMMythicCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	l := h.Logger.With(
		"command", "topm",
		"guild_id", i.GuildID,
		"user", i.Member.User.Username,
	)

	l.Info("generating mythic guild ranking")

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	subs, err := h.SubStore.GetGuildSubsByChannel(i.ChannelID)
	if err != nil || len(subs) == 0 {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: pointer("❌ В этом канале не настроено отслеживание гильдий."),
		})
		l.Error("empty guilds tracked list", "error", err)
		return
	}
	targetSub := subs[0]

	guids, err := h.GMStore.GetPlayersByGuildID(targetSub.Realm, targetSub.GuildID)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: pointer("❌ Ошибка при получении участников гильдии."),
		})
		l.Error("getting guild members error", "error", err)
		return
	}
	if len(guids) == 0 {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: pointer("📭 В базе нет данных об участниках этой гильдии."),
		})
		return
	}

	players, err := h.MythicStore.GetGuildMythicPlayers(targetSub.Realm, guids)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: pointer("❌ Ошибка при получении мифик-рейтинга."),
		})
		l.Error("getting guild mythic players error", "error", err)
		return
	}
	if len(players) == 0 {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: pointer("📭 Мифик-рейтинг отсутствует."),
		})
		l.Info("empty mythic leaderboard for guild")
		return
	}

	guildName := "Рейтинг гильдии"
	if n, err := h.SubStore.GetGuildName(targetSub.GuildID, targetSub.Realm); err == nil && n != "" {
		guildName = n
	}

	img, err := RenderMythicGuildTopImage(guildName, targetSub.Realm, players)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: pointer("❌ Ошибка при генерации изображения."),
		})
		l.Error("mythic top image generation error", "error", err)
		return
	}
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Files: []*discordgo.File{
			{
				Name:   "topm.png",
				Reader: bytes.NewReader(img),
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

	channelID := i.ChannelID
	switch data.Name {
	case "menu":
		h.HandleMenuCommand(s, i)
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
	case "top":
		h.HandleTopMCommand(s, i)

	case "topm":
		h.HandleTopMMythicCommand(s, i)

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
						Description: "**⚠️ Не приходят отчёты?** Проверьте, что у бота есть права **Просмотр канала** и **Отправка сообщений** в этом канале. Без них бот не сможет писать сюда.\n\n" +

							"*Настройка бота:*\n" +
							"1. Создайте текстовый канал (например, #raid-alerts) и *выдайте боту права на чтение и отправку сообщений в него*.\n" +
							"2. В этом канале введите /menu — откроется меню с кнопками.\n" +
							"3. Нажмите «🏰 Добавить гильдию», выберите сервер и введите ID гильдии. Аналогично можно добавить игрока через «👤 Добавить игрока».\n" +
							"Бот сразу начнёт мониторинг. Если отчёт отправился — всё работает, можно в /menu отключить лишние типы отчётов.\n" +

							"*Команды:*" +
							"**/menu** Вызывает меню с настройкой подписок бота, трекинга и т.д.\n" +
							"**/top** Открывает меню с кнопками, нажав на которые можно отправить рейтинги по чек боссам среди игроков гильдии.\n" +
							"**/topm** Показывает мифик-рейтинг (Рио) среди участников гильдии — таблица с Илвл, зодиаком, чёрными бриллиантами и лучшим ключом.\n" +
							"**/list** — Список отслеживаемых гильдий в данном канале.\n" +
							"**/listcats** — Список отслеживаемых игроков в данном канале.\n" +
							"**/help** — Показать это сообщение.\n" +
							"Бот может следить как за гильдиями так и за конкретными игроками. " +
							"Например  в гильдии есть статик десятки смешанный с пугами, и сирус не засчитывает их килы как гильдейские. " +
							"Вы можете отслеживать РЛа этого статика по его нику и бот будет присылать отчеты килов этого статика.",
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
		h.Logger.Info("empty channel ID")
		return
	}

	// Remember the welcome channel so we can notify the guild later (e.g. when
	// the bot has no permission to post reports to a subscribed channel).
	if err := h.SubStore.SetWelcomeChannel(g.ID, channelID); err != nil {
		h.Logger.Error("Save welcome channel err", "error", err, "discord_guild_id", g.ID)
	}

	if h.SubStore.IsDiscordGuildSubscribed(g.ID) {
		h.Logger.Info("guild already subscribed", "discord_guild_id", g.ID)
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "🐈 Привет! Я LazyCatBot",
		Description: "Я помогу вам отслеживать убийства боссов вашей гильдии!\n\n" +
			"**Как меня настроить:**\n" +
			"Создайте текстовый канал и введите в этом канале команду /menu\n\n" +
			"Появится меню с возможностью добавить гильдию для отслеживания по ID гильдии.\n\n" +
			"**⚠️ Важно:** чтобы я мог присылать отчёты, выдайте мне права на текстовые каналы — " +
			"**Просмотр канала** и **Отправка сообщений** (View Channel и Send Messages). " +
			"Без этих прав отчёты не будут доставляться.\n\n" +
			"Список комманд /help\n\n" +
			"*(ID гильдии можно найти в ссылке на вашу гильдию на сайте Sirus)*",
		Color: 0xf1c40f,
	}

	s.ChannelMessageSendEmbed(channelID, embed)
}
