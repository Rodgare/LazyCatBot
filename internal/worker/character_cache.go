package worker

import (
	"context"
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

		candidates := map[string]struct{}{}
		if players, err := w.pSubStore.GetAllPlayers(); err == nil {
			for _, p := range players {
				candidates[p.Realm+"|"+p.Name] = struct{}{}
			}
		} else {
			w.logger.Error("[CharacterCacheUpdater] Get players err", "error", err)
		}
		if members, err := w.gmStore.GetAllMembers(); err == nil {
			for _, m := range members {
				candidates[m.Realm+"|"+m.Name] = struct{}{}
			}
		} else {
			w.logger.Error("[CharacterCacheUpdater] Get members err", "error", err)
		}

		refreshed := 0
		for key := range candidates {
			if refreshed >= maxPerCycle {
				break
			}

			parts := strings.SplitN(key, "|", 2)
			realm, name := parts[0], parts[1]

			stale, err := w.charStore.StaleOrMissing(realm, name, ttl)
			if err != nil || !stale {
				continue
			}

			w.apiLimiter.Wait()
			data, err := w.sirusClient.FetchCharacter(realm, name)
			if err != nil {
				w.logger.Error("[CharacterCacheUpdater] Fetch character err", "error", err, "realm", realm, "name", name)
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