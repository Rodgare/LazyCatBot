package worker

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// PriorityChar is a character that should be refreshed in the cache before the
// regular TTL sweep. These are usually raid/mythic participants that showed up
// in a kill without an existing subscription.
type PriorityChar struct {
	Realm string
	ID    int
	Name  string
}

// EnqueueCharacterRefresh queues a character for an early cache refresh. It is
// non-blocking; if the pool is full the request is dropped and logged.
func (w *Worker) EnqueueCharacterRefresh(realm string, id int, name string) {
	select {
	case w.charPriority <- PriorityChar{Realm: realm, ID: id, Name: name}:
	default:
		w.logger.Warn("[CharacterCache] Priority pool full, skipping refresh", "realm", realm, "name", name)
	}
}

// enqueueCharRefreshIfStale queues a character for refresh only when its cached
// record is missing or older than the TTL, so freshly cached players are not
// re-fetched after every kill.
func (w *Worker) enqueueCharRefreshIfStale(realm, name string, id int) {
	const ttl = 24 * 60 * 60
	stale, err := w.charStore.StaleOrMissing(realm, name, ttl)
	if err != nil || !stale {
		return
	}
	w.EnqueueCharacterRefresh(realm, id, name)
}

func (w *Worker) CharacterCacheUpdater(ctx context.Context) {
	const ttl = 24 * 60 * 60
	const maxPerCycle = 50
	const cycle = 15 * time.Minute
	const priorityWait = 10 * time.Second

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("[CharacterCacheUpdater] Processor stopped")
			return
		default:
		}

		refreshed := 0

		// 1. Priority pool first: participants seen in recent kills get refreshed
		// before the regular TTL sweep.
		priority := make([]PriorityChar, 0, maxPerCycle)
		for len(priority) < maxPerCycle {
			select {
			case pc := <-w.charPriority:
				priority = append(priority, pc)
			default:
				goto priorityDone
			}
		}
	priorityDone:
		for _, pc := range priority {
			if refreshed >= maxPerCycle {
				break
			}
			stale, err := w.charStore.StaleOrMissing(pc.Realm, pc.Name, ttl)
			if err != nil || !stale {
				continue
			}

			w.apiLimiter.Wait()
			data, err := w.sirusClient.FetchCharacter(pc.Realm, pc.ID)
			if err != nil {
				w.logger.Error("[CharacterCacheUpdater] Fetch character err", "error", err, "realm", pc.Realm, "name", pc.Name, "id", pc.ID)
				continue
			}

			if err := w.charStore.Upsert(pc.Realm, pc.Name, *data); err != nil {
				w.logger.Error("[CharacterCacheUpdater] Upsert character err", "error", err, "realm", pc.Realm, "name", pc.Name)
				continue
			}
			refreshed++
		}

		// 2. Regular TTL candidates fill the remaining budget of this cycle.
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

		// When the priority pool still has work, loop again soon; otherwise
		// return to the slow TTL cycle.
		if len(w.charPriority) > 0 {
			select {
			case <-ctx.Done():
				w.logger.Info("[CharacterCacheUpdater] Processor stopped")
				return
			case <-time.After(priorityWait):
			}
		} else {
			select {
			case <-ctx.Done():
				w.logger.Info("[CharacterCacheUpdater] Processor stopped")
				return
			case <-time.After(cycle):
			}
		}
	}
}
