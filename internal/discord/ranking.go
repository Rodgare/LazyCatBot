package discord

import (
	"LazyCatBot/internal/models"
	"LazyCatBot/internal/sirus"
	"bytes"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (h *BotHandler) SendBossRanking(s *discordgo.Session, i *discordgo.InteractionCreate, raidID, bossID int, role string) {
	l := h.Logger.With(
		"raid_id", raidID,
		"boss_id", bossID,
		"role", role,
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

	players, err := h.LbStore.GetBossTop(targetSub.Realm, raidID, bossID, targetSub.GuildID, role)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: pointer("❌ Ошибка при получении данных из базы."),
		})
		l.Error("Getting data from db error", "error", err)
		return
	}

	if len(players) == 0 {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: pointer("📭 Данных по этому боссу пока нет."),
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

	report := models.BossKillReport{
		MapName:   "Рейтинг гильдии",
		BossName:  fmt.Sprintf("%s (%s)", sirus.GetBossName(raidID, bossID), strings.ToUpper(role)),
		RaidOrder: raidID,
		Players:   players,
		Realm:     targetSub.Realm,
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
