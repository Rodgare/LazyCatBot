package sirus

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func GetPage(raidID, bossID int, page int) (*Leaderboard, error) {
	weekFrom, weekTo := GetSirusDates()
	url := fmt.Sprintf("https://sirus.su/api/base/x3/leaderboard/pve?ladder=players&type=dps&aggregation=max&week_from=%s&week_to=%s&i=%d&boss=%d&page=%d",
		weekFrom, weekTo, raidID, bossID, page)

	var res Leaderboard
	if err := makeRequest(url, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func FetchFullLeaderboard(raidID, bossID int) ([]LeaderboardPlayer, error) {
	var allPlayers []LeaderboardPlayer

	firstPage, err := GetPage(raidID, bossID, 1)
	if err != nil {
		return nil, err
	}
	time.Sleep(5 * time.Second)
	allPlayers = append(allPlayers, firstPage.Data...)
	totalPages := firstPage.Meta.LastPage

	for p := 2; p <= totalPages; p++ {
		nextPage, err := GetPage(raidID, bossID, p)
		if err != nil {
			fmt.Printf("Error loading page %d: %v\n", p, err)
			time.Sleep(5 * time.Second)
			continue
		}
		allPlayers = append(allPlayers, nextPage.Data...)
		time.Sleep(5 * time.Second)
	}

	return allPlayers, nil
}

func FetchActualRaids() (ActualRaids, error) {
	url := "https://sirus.su/api/base/22/progression/pve/realm-progress"
	var res ActualRaids

	if err := makeRequest(url, &res); err != nil {
		return nil, err
	}

	return res, nil
}

func FetchGuildLatestBossKills(page int, guildID int) (*LatestBossKills, error) {
	weekFrom, weekTo := GetSirusDates()
	url := fmt.Sprintf("https://sirus.su/api/base/22/progression/pve/latest-boss-kills?page=%d&week_from=%s&week_to=%s&guild=%d", page, weekFrom, weekTo, guildID)
	var res LatestBossKills

	if err := makeRequest(url, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func FetchBossFightDetails(fightID int) (*BossFight, error) {
	url := fmt.Sprintf("https://sirus.su/api/base/22/details/bossfight/%v", fightID)
	var fight BossFight

	if err := makeRequest(url, &fight); err != nil {
		return nil, err
	}

	return &fight, nil
}

func GetSirusDates() (string, string) {
	loc := time.FixedZone("MSK", 3*3600)
	return calculateSirusDates(time.Now().In(loc))
}

func calculateSirusDates(now time.Time) (string, string) {
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

func makeRequest(url string, target any) error {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) LazyCatBot/1.0")
	req.Header.Set("Accept", "application/json")

	var lastErr error
	for try := 1; try <= 3; try++ {
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("Attempt %d failed (network error): %v", try, err)
			time.Sleep(2 * time.Second)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("API returned status: %d", resp.StatusCode)
			resp.Body.Close()
			log.Printf("Attempt %d failed (status %d)", try, resp.StatusCode)
			time.Sleep(2 * time.Second)
			continue
		}
		defer resp.Body.Close()
		return json.NewDecoder(resp.Body).Decode(target)
	}

	return fmt.Errorf("all request attempts failed: %w", lastErr)
}
