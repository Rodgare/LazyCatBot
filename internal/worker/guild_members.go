package worker

import (
	"time"
)

func (w *Worker) GuildMembersUpdater() {
	for {
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
				time.Sleep(1 * time.Minute)
			}

			if gms != nil {
				err = w.gmStore.UpdateGuildMembers(gID, *gms)
				if err != nil {
					w.logger.Error("Update Guild Members error", "error", err, "guild_id", gID)
				}
			}

			time.Sleep(5 * time.Second)
		}

		time.Sleep(1 * time.Hour)
	}
}
