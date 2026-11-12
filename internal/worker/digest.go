package worker

import (
	"fmt"
	"time"
)

// SendWeeklyDigests builds and posts a weekly image for every premium channel
// that tracks a guild. Free channels are skipped; the digest is a premium
// feature. It is safe to call manually (e.g. to test) and from the cron.
func (w *Worker) SendWeeklyDigests() {
	if w.killHistory == nil {
		w.logger.Warn("[Digest] Kill history store is not wired, skipping")
		return
	}

	guilds, err := w.subStore.GetTrackedGuilds()
	if err != nil {
		w.logger.Error("[Digest] GetTrackedGuilds err", "error", err)
		return
	}

	now := time.Now()
	since := now.AddDate(0, 0, -7).Unix()
	until := now.Unix()

	sent := 0
	for key, channels := range guilds {
		digest, err := w.killHistory.GetWeeklyDigest(key.Realm, key.GuildID, since, until)
		if err != nil {
			w.logger.Error("[Digest] GetWeeklyDigest err", "error", err,
				"realm", key.Realm, "guild_id", key.GuildID)
			continue
		}

		if w.gmStore != nil && w.mythLbStore != nil {
			if guids, err := w.gmStore.GetPlayersByGuildID(key.Realm, key.GuildID); err == nil && len(guids) > 0 {
				if mythic, err := w.mythLbStore.GetGuildMythicPlayers(key.Realm, guids); err == nil {
					if len(mythic) > 5 {
						mythic = mythic[:5]
					}
					digest.MythicTop = mythic
				} else {
					w.logger.Error("[Digest] GetGuildMythicPlayers err", "error", err,
						"realm", key.Realm, "guild_id", key.GuildID)
				}
			}
		}

		if digest.TotalKills == 0 && len(digest.MythicTop) == 0 {
			continue
		}

		if name, err := w.subStore.GetGuildName(key.GuildID, key.Realm); err == nil && name != "" {
			digest.GuildName = name
		}
		if digest.GuildName == "" {
			digest.GuildName = fmt.Sprintf("Гильдия %d", key.GuildID)
		}

		for _, ch := range channels {
			if w.premiumGate != nil && !w.premiumGate.IsPremium(ch) {
				continue
			}
			if err := w.reporter.SendWeeklyDigest(ch, *digest); err != nil {
				w.logger.Error("[Digest] SendWeeklyDigest err", "error", err,
					"channel", ch, "realm", key.Realm, "guild_id", key.GuildID)
				continue
			}
			sent++
			time.Sleep(1 * time.Second)
		}
	}

	w.logger.Info("[Digest] Weekly digests finished", "sent", sent)
}
