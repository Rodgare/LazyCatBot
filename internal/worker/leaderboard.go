package worker

import (
	"LazyCatBot/internal/sirus"
	"LazyCatBot/internal/storage"
	"fmt"
	"log"
	"time"
)

// raid order: boss/encounter order
var RaidsToFetch = map[int][]int{
	13: {0},
	15: {0, 1, 2, 3, 4},
	16: {0},
	17: {0, 1},
}

// class order: spec order
var ClassesToFetch = map[int][]int{
	1:  {0, 1, 2},
	2:  {0, 1, 2},
	3:  {0, 1, 2},
	4:  {0, 1, 2},
	5:  {0, 1, 2},
	6:  {0, 1, 2},
	7:  {0, 1, 2},
	8:  {0, 1, 2},
	9:  {0, 1, 2},
	11: {0, 1, 2},
}

func StartLeaderboardSync(store *storage.LeaderboardStorage) {
	fmt.Println("[Worker] Запуск системы синхронизации...")

	for {
		fmt.Println("[Worker] Начало цикла обновления данных...")

		for raidOrder, bosses := range RaidsToFetch {
			for _, bossOrder := range bosses {
				for class, specs := range ClassesToFetch {
					for _, spec := range specs {
						fmt.Printf("[Worker] Parsing (R:%v, B:%v, C:%v, S:%v)...\n", raidOrder, bossOrder, class, spec)

						classSpec := fmt.Sprintf("%d:%d", class, spec)
						players, err := sirus.FetchFullLeaderboard(raidOrder, bossOrder, classSpec)
						if err != nil {
							log.Printf("[Worker] Error (R:%d B:%d C:%d S:%d): %v", raidOrder, bossOrder, class, spec, err)
							time.Sleep(2 * time.Second)
							continue
						}
						if len(players) == 0 {
							fmt.Printf("[Worker] Empty players data R:%d B:%d C:%d S:%d\n", raidOrder, bossOrder, class, spec)
							continue
						}

						store.UpdateLeaderboardStorage(raidOrder, bossOrder, class, spec, players)

						time.Sleep(2 * time.Second)
					}

				}
			}
		}

		fmt.Println("[Worker] Все данные обновлены. Спим 60 минут...")
		time.Sleep(60 * time.Minute)
	}
}
