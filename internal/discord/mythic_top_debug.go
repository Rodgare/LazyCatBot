package discord

import (
	"LazyCatBot/internal/models"
	"bytes"
	"encoding/json"
	"log/slog"
	"os"

	"github.com/bwmarrin/discordgo"
)

// SendMythicTopMock renders the guild mythic leaderboard table from
// mock/mythic_top.json and sends it to the given channel. Used for debug.
func (r *DiscordReporter) SendMythicTopMock(channelID string) error {
	data, err := os.ReadFile("mock/mythic_top.json")
	if err != nil {
		slog.Error("Mock mythic top json read error", "error", err)
		return err
	}

	var players []models.GuildMythicPlayer
	if err := json.Unmarshal(data, &players); err != nil {
		slog.Error("Mock mythic top json unmarshal error", "error", err)
		return err
	}

	img, err := RenderMythicGuildTopImage("Тестовая гильдия", "x3", players)
	if err != nil {
		slog.Error("Mock mythic top render error", "error", err)
		return err
	}

	params := &discordgo.MessageSend{
		Files: []*discordgo.File{
			{
				Name:   "topm.png",
				Reader: bytes.NewReader(img),
			},
		},
	}
	_, err = r.session.ChannelMessageSendComplex(channelID, params)
	if err != nil {
		slog.Error("Error sending mythic top mock", "error", err, "channel_id", channelID)
	}
	return err
}