package worker

import (
	"LazyCatBot/internal/sirus"
	"LazyCatBot/internal/storage"
	"log"
	"time"
)

func GuildMembersUpdater(subStore *storage.SubscribeStorage, gmStore *storage.GuildMembersStorage) {
	for {
		guildsToUpdate, err := subStore.GetTrackedGuilds()
		if err != nil {
			log.Printf("[GuildMembersUpdater] Getting tracked guilds err: %v\n", err)
		}

		gIDs := make([]int, 0, len(guildsToUpdate))
		for k := range guildsToUpdate {
			gIDs = append(gIDs, k)
		}

		for _, gID := range gIDs {
			gms, err := sirus.FetchGuildMembers(gID)
			if err != nil {
				log.Printf("[GuildMembersUpdater] FetchGuildMembers error, guild ID: %d | err: %v\n", gID, err)
				time.Sleep(1 * time.Minute)
			}

			if gms != nil {
				err = gmStore.UpdateGuildMembers(gID, *gms)
				if err != nil {
					log.Printf("[GuildMembersUpdater] Update Guild Members error, guild ID: %d | err: %v\n", gID, err)
				}
			}

			time.Sleep(5 * time.Second)
		}

		time.Sleep(1 * time.Hour)
	}
}
