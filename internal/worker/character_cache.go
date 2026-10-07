package worker

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (w *Worker) CharacterCacheUpdater(ctx context.Context) {
	const ttl = 24 * 60 * 60
	const maxPerCycle = 50
	const cycle = 15 * time.Minute

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("[CharacterCacheUpdater] Processor stopped")
			return
		default:
		}

		candidates := map[string]string{}
		if players, err := w.pSubStore.GetAllPlayers(); err == nil {
			for _, p := range players {
				candidates[fmt.Sprintf("%d|%s", p.ID, p.Realm)] = p.Name
			}
		} else {
			w.logger.Error("[CharacterCacheUpdater] Get players err", "error", err)
		}
		if members, err := w.gmStore.GetAllMembers(); err == nil {
			for _, m := range members {
				candidates[fmt.Sprintf("%d|%s", m.ID, m.Realm)] = m.Name
			}
		} else {
			w.logger.Error("[CharacterCacheUpdater] Get members err", "error", err)
		}

		refreshed := 0
		for key, name := range candidates {
			if refreshed >= maxPerCycle {
				break
			}

			parts := strings.SplitN(key, "|", 2)
			realm := parts[1]
			var id int
			fmt.Sscanf(parts[0], "%d", &id)

			stale, err := w.charStore.StaleOrMissing(realm, name, ttl)
			if err != nil || !stale {
				continue
			}

			w.apiLimiter.Wait()
			data, err := w.sirusClient.FetchCharacter(realm, id)
			if err != nil {
				w.logger.Error("[CharacterCacheUpdater] Fetch character err", "error", err, "realm", realm, "name", name, "id", id)
				continue
			}

			if err := w.charStore.Upsert(realm, name, *data); err != nil {
				w.logger.Error("[CharacterCacheUpdater] Upsert character err", "error", err, "realm", realm, "name", name)
				continue
			}
			refreshed++
		}

		select {
		case <-ctx.Done():
			w.logger.Info("[CharacterCacheUpdater] Processor stopped")
			return
		case <-time.After(cycle):
		}
	}
}