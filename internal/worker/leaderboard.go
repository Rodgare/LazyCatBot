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

func StartLeaderboardSync(store *storage.LeaderboardStorage) {
	fmt.Println("[Worker] Запуск системы синхронизации...")

	for {
		fmt.Println("[Worker] Начало цикла обновления данных...")

		for raid, bosses := range RaidsToFetch {
			for _, boss := range bosses {
				fmt.Printf("[Worker] Parsing (R:%v, B:%v)...\n", raid, boss)

				players, err := sirus.FetchFullLeaderboard(raid, boss)

				if err != nil {
					log.Printf("[Worker] Error (R:%d B:%d): %v", raid, boss, err)
					time.Sleep(2 * time.Second)
					continue
				}

				if len(players) == 0 {
					fmt.Printf("[Worker] Empty players data R:%d B:%d\n", raid, boss)
					continue
				}

				err = store.UpdateLeaderboardStorage(raid, boss, players)
				if err != nil {
					log.Printf("[Worker] Ошибка сохранения в базу (R:%d B:%d): %v", raid, boss, err)
				}

				time.Sleep(2 * time.Second)
			}
		}

		fmt.Println("[Worker] Все данные обновлены. Спим 12 часов...")
		time.Sleep(12 * time.Hour)
	}
}
