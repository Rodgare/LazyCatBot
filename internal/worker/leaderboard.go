package worker

import (
	"LazyCatBot/internal/sirus"
	"LazyCatBot/internal/storage"
	"fmt"
	"log"
	"time"
)

var RaidsToFetch = map[int][]int{
	13: {0},
	15: {0, 1, 2, 3, 4},
	16: {0},
	17: {0, 1},
}
var SpecsToFetch = []string{"1:0", "1:1", "1:2", "2:0", "2:1", "2:2", "3:0", "3:1", "3:2", "4:0", "4:1", "4:2",
	"5:0", "5:1", "5:2", "6:0", "6:1", "6:2", "7:0", "7:1", "7:2", "8:0", "8:1", "8:2", "9:0", "9:1", "9:2", "11:0", "11:1", "11:2"}

	func StartLeaderboardSync(store *storage.LeaderboardStorage) {
		fmt.Println("[Worker] Запуск системы синхронизации...")
	
		for {
			fmt.Println("[Worker] Начало цикла обновления данных...")
	
			for raidID, bosses := range RaidsToFetch {
				for _, bossID := range bosses {
					for _, spec := range SpecsToFetch {
						
						players, err := sirus.FetchFullLeaderboard(raidID, bossID, spec)
						if err != nil {
							log.Printf("[Worker] Ошибка (R:%d B:%d S:%s): %v", raidID, bossID, spec, err)
							time.Sleep(2 * time.Second)
							continue
						}
	
						store.Update(raidID, bossID, spec, players)
						
						time.Sleep(2 * time.Second)
					}
				}
			}
	
			fmt.Println("[Worker] Все данные обновлены. Спим 60 минут...")
			time.Sleep(60 * time.Minute)
		}
	}