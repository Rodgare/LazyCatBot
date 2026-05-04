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
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	guilds, err := h.SubStore.GetGuildsByChannel(i.ChannelID)
	if err != nil || len(guilds) == 0 {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: pointer("❌ В этом канале не настроено отслеживание гильдий."),
		})
		return
	}

	targetGuildID := guilds[0]

	players, err := h.LbStore.GetBossTop(raidID, bossID, targetGuildID, role)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: pointer("❌ Ошибка при получении данных из базы."),
		})
		return
	}

	if len(players) == 0 {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: pointer("📭 Данных по этому боссу пока нет."),
		})
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
	}
	img, err := RenderReportImage(report)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: pointer("❌ Ошибка при генерации изображения."),
		})
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
