package worker

import (
	"context"
	"time"
)

func (w *Worker) GuildMembersUpdater(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("[GuildMembersUpdater] Processor stopped")
			return
		default:
		}

		guildsToUpdate, err := w.subStore.GetTrackedGuilds()
		if err != nil {
			w.logger.Error("Getting tracked guilds err", "error", err)
		}

		for key := range guildsToUpdate {
			gms, err := w.sirusClient.FetchGuildMembers(key.Realm, key.GuildID)
			if err != nil {
				w.logger.Error("FetchGuildMembers error", "error", err, "guild_id", key.GuildID, "realm", key.Realm)
				select {
				case <-ctx.Done():
					w.logger.Info("[GuildMembersUpdater] Processor stopped")
					return
				case <-time.After(1 * time.Minute):
				}
				continue
			}

			if gms != nil {
				err = w.gmStore.UpdateGuildMembers(key.Realm, key.GuildID, *gms)
				if err != nil {
					w.logger.Error("Update Guild Members error", "error", err, "guild_id", key.GuildID, "realm", key.Realm)
				}
			}

			select {
			case <-ctx.Done():
				w.logger.Info("[GuildMembersUpdater] Processor stopped")
				return
			case <-time.After(5 * time.Second):
			}
		}

		select {
		case <-ctx.Done():
			w.logger.Info("[GuildMembersUpdater] Processor stopped")
			return
		case <-time.After(1 * time.Hour):
		}
	}
}
