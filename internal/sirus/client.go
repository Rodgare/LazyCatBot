package sirus

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func GetPage(raidID, bossID int, spec string, page int) (*Leaderboard, error) {
    weekFrom, weekTo := GetSirusDates()
    url := fmt.Sprintf("https://sirus.su/api/base/x3/leaderboard/pve?ladder=players&type=dps&aggregation=max&week_from=%s&week_to=%s&i=%d&boss=%d&specs=%s&page=%d",
        weekFrom, weekTo, raidID, bossID, spec, page)

    httpClient := &http.Client{
        Timeout: 10 * time.Second,
    }

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }

    req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) LazyCatBot/1.0")
    req.Header.Set("Accept", "application/json")

    resp, err := httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("API вернуло статус: %d", resp.StatusCode)
    }

    var res Leaderboard
    if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
        return nil, err
    }

    return &res, nil
}

func FetchFullLeaderboard(raidID, bossID int, spec string) ([]LeaderboardPlayer, error) {
	var allPlayers []LeaderboardPlayer

	firstPage, err := GetPage(raidID, bossID, spec, 1)
	if err != nil {
		return nil, err
	}

	allPlayers = append(allPlayers, firstPage.Data...)
	totalPages := firstPage.Meta.LastPage

	for p := 2; p <= totalPages; p++ {
		time.Sleep(300 * time.Millisecond)

		nextPage, err := GetPage(raidID, bossID, spec, p)
		if err != nil {
			fmt.Printf("Ошибка при загрузке страницы %d: %v\n", p, err)
			continue
		}
		allPlayers = append(allPlayers, nextPage.Data...)
	}

	return allPlayers, nil
}

func GetSirusDates() (string, string) {
	now := time.Now()

	daysSinceThursday := int(now.Weekday()) - int(time.Thursday)
	if daysSinceThursday < 0 {
		daysSinceThursday += 7
	}

	weekTo := now.Format("2006-01-02")

	lastThursday := now.AddDate(0, 0, -daysSinceThursday)
	weekFromDate := lastThursday.AddDate(0, 0, -28)
	weekFrom := weekFromDate.Format("2006-01-02")

	return weekFrom, weekTo
}