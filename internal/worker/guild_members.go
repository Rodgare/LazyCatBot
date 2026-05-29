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

		gIDs := make([]int, 0, len(guildsToUpdate))
		for k := range guildsToUpdate {
			gIDs = append(gIDs, k)
		}

		for _, gID := range gIDs {
			gms, err := w.sirusClient.FetchGuildMembers(gID)
			if err != nil {
				w.logger.Error("FetchGuildMembers error", "error", err, "guild_id", gID)
				select {
				case <-ctx.Done():
					w.logger.Info("[GuildMembersUpdater] Processor stopped")
					return
				case <-time.After(1 * time.Minute):
				}
			}

			if gms != nil {
				err = w.gmStore.UpdateGuildMembers(gID, *gms)
				if err != nil {
					w.logger.Error("Update Guild Members error", "error", err, "guild_id", gID)
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
