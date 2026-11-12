package discord

import (
	"LazyCatBot/internal/models"
	"LazyCatBot/internal/sirus"
	"LazyCatBot/internal/storage"
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (h *BotHandler) SendBossRanking(s *discordgo.Session, i *discordgo.InteractionCreate, raidID, bossID int, role string, rng storage.LeaderboardRange) {
	if !rng.Valid() {
		rng = storage.Range2W
	}
	l := h.Logger.With(
		"raid_id", raidID,
		"boss_id", bossID,
		"role", role,
		"range", rng,
		"guild_id", i.GuildID,
	)

	l.Info("generating boss ranking report")

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

	players, err := h.LbStore.GetBossTopRange(targetSub.Realm, raidID, bossID, targetSub.GuildID, role, rng)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: pointer("❌ Ошибка при получении данных из базы."),
		})
		l.Error("Getting data from db error", "error", err)
		return
	}

	if len(players) == 0 {
		msg := "📭 Данных по этому боссу пока нет."
		if rng != storage.Range2W {
			msg = fmt.Sprintf("📭 Для диапазона «%s» пока нет накопленных данных. Он собирается автоматически с ночными синками.", rng.Title())
		}
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: pointer(msg),
		})
		l.Info("Empty boss kil data")
		return
	}

	for j := range players {
		p, err := h.LbStore.GetPlayerRank(raidID, bossID, players[j], role)
		if err == nil {
			players[j] = p
		}
	}

	guildName := "Рейтинг гильдии"
	if n, err := h.SubStore.GetGuildName(targetSub.GuildID, targetSub.Realm); err == nil && n != "" {
		guildName = n
	}

	report := models.BossKillReport{
		MapName:    guildName,
		BossName:   fmt.Sprintf("%s (%s · %s)", sirus.GetBossName(raidID, bossID), strings.ToUpper(role), rng.Title()),
		RaidOrder:  raidID,
		Players:    players,
		Realm:      targetSub.Realm,
		RangeLabel: rng.Title(),
	}
	img, err := RenderReportImage(report)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: pointer("❌ Ошибка при генерации изображения."),
		})
		l.Error("Image geenration error", "error", err)
		return
	}
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Files: []*discordgo.File{
			{
				Name:   "ranking.png",
				Reader: bytes.NewReader(img),
			},
		},
	})

}

func pointer(s string) *string { return &s }

// handleRankingRangePrompt asks which time window the user wants before
// rendering a boss ranking. 2 KD is free; 4 KD and all-time require premium.
func (h *BotHandler) handleRankingRangePrompt(s *discordgo.Session, i *discordgo.InteractionCreate, raidID, bossID int, role string) {
	bossName := sirus.GetBossName(raidID, bossID)
	if bossName == "" {
		bossName = fmt.Sprintf("Босс %d", bossID)
	}

	rangeButton := func(label string, rng storage.LeaderboardRange, style discordgo.ButtonStyle) discordgo.MessageComponent {
		return discordgo.Button{
			Label:    label,
			Style:    style,
			CustomID: fmt.Sprintf("range_%s_%s_%d_%d", rng, role, raidID, bossID),
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("🏆 **%s · %s**\nВыберите период рейтинга:", bossName, strings.ToUpper(role)),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					rangeButton("📅 2 КД", storage.Range2W, discordgo.SecondaryButton),
					rangeButton("📅 4 КД", storage.Range4W, discordgo.PrimaryButton),
					rangeButton("♾ Всё время", storage.RangeAll, discordgo.PrimaryButton),
				}},
			},
		},
	})
}

func (h *BotHandler) handleRankingRangeButton(s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	parts := strings.Split(customID, "_")
	if len(parts) != 5 {
		return
	}
	rng := storage.LeaderboardRange(parts[1])
	if !rng.Valid() {
		return
	}
	role := parts[2]
	if role != "dps" && role != "hps" {
		return
	}
	raidID, errRaid := strconv.Atoi(parts[3])
	bossID, errBoss := strconv.Atoi(parts[4])
	if errRaid != nil || errBoss != nil {
		return
	}

	if rng != storage.Range2W {
		premium := h.PremiumStore != nil && h.PremiumStore.IsPremium(i.GuildID)
		if !premium {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("🔒 Рейтинг «%s» доступен с премиумом для всего сервера.\n"+
						"Оформите его командой `/premium` — код активации выдаётся сразу.", rng.Title()),
					Flags: discordgo.MessageFlagsEphemeral,
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{Components: []discordgo.MessageComponent{
							discordgo.Button{Label: "🐾 Оформить премиум", Style: discordgo.PrimaryButton, CustomID: "premium_open"},
						}},
					},
				},
			})
			return
		}
	}

	h.SendBossRanking(s, i, raidID, bossID, role, rng)
}
