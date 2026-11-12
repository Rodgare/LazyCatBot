package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func (h *BotHandler) isPremiumChannel(i *discordgo.InteractionCreate) bool {
	return h.PremiumStore != nil && h.PremiumStore.IsPremium(i.GuildID)
}

func (h *BotHandler) freeLimitUpsell(s *discordgo.Session, i *discordgo.InteractionCreate, reason string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "🔒 " + reason + "\n\nПремиум для сервера снимает лимиты во всех каналах: `/premium`.",
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{Label: "🐾 Оформить премиум", Style: discordgo.PrimaryButton, CustomID: "premium_open"},
				}},
			},
		},
	})
}

func hasRealmConflict(realms []string, realm string) bool {
	for _, r := range realms {
		if r != "" && r != realm {
			return true
		}
	}
	return false
}

// canAddGuild enforces free-tier limits. Premium servers are unlimited, and
// re-adding an already tracked guild is always allowed.
func (h *BotHandler) canAddGuild(i *discordgo.InteractionCreate, guildID int, realm string) (bool, string) {
	if h.isPremiumChannel(i) {
		return true, ""
	}
	if h.SubStore.HasGuild(guildID, i.ChannelID) {
		return true, ""
	}

	guildSubs, err := h.SubStore.GetGuildSubsByChannel(i.ChannelID)
	if err != nil {
		return false, "Не удалось проверить лимиты канала. Попробуйте позже."
	}
	playerSubs, err := h.PlayerSubStore.GetPlayerSubsByChannel(i.ChannelID)
	if err != nil {
		return false, "Не удалось проверить лимиты канала. Попробуйте позже."
	}

	realms := make([]string, 0, len(guildSubs)+len(playerSubs))
	for _, g := range guildSubs {
		realms = append(realms, g.Realm)
	}
	for _, p := range playerSubs {
		realms = append(realms, p.Realm)
	}
	if hasRealmConflict(realms, realm) {
		return false, "На бесплатном тарифе все гильдии и игроки канала должны быть с одного реалма. Премиум позволяет отслеживать несколько реалмов."
	}
	if h.FreeGuildLimit > 0 && len(guildSubs) >= h.FreeGuildLimit {
		return false, fmt.Sprintf("На бесплатном тарифе можно отслеживать только %d гильдию в канале.", h.FreeGuildLimit)
	}
	return true, ""
}

// canAddPlayer enforces free-tier limits for player subscriptions. Premium is\n// guild-wide, so any channel of a premium server is unconstrained.
func (h *BotHandler) canAddPlayer(i *discordgo.InteractionCreate, playerID int, realm string) (bool, string) {
	if h.isPremiumChannel(i) {
		return true, ""
	}
	if h.PlayerSubStore.HasPlayer(playerID, i.ChannelID) {
		return true, ""
	}

	guildSubs, err := h.SubStore.GetGuildSubsByChannel(i.ChannelID)
	if err != nil {
		return false, "Не удалось проверить лимиты канала. Попробуйте позже."
	}
	playerSubs, err := h.PlayerSubStore.GetPlayerSubsByChannel(i.ChannelID)
	if err != nil {
		return false, "Не удалось проверить лимиты канала. Попробуйте позже."
	}

	realms := make([]string, 0, len(guildSubs)+len(playerSubs))
	for _, g := range guildSubs {
		realms = append(realms, g.Realm)
	}
	for _, p := range playerSubs {
		realms = append(realms, p.Realm)
	}
	if hasRealmConflict(realms, realm) {
		return false, "На бесплатном тарифе все гильдии и игроки канала должны быть с одного реалма. Премиум позволяет отслеживать несколько реалмов."
	}
	if h.FreePlayerLimit > 0 && len(playerSubs) >= h.FreePlayerLimit {
		return false, fmt.Sprintf("На бесплатном тарифе можно отслеживать до %d игроков в канале.", h.FreePlayerLimit)
	}
	return true, ""
}
