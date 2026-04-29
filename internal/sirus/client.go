package sirus

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"context"

	"github.com/chromedp/chromedp"
)

func GetPage(raidID, bossID, classID, specID int, role string, page int) (*Leaderboard, error) {
	fmt.Printf("[Worker] Pasing page %d R: %d B: %d classID: %d, specID: %d\n", page, raidID, bossID, classID, specID)
	weekFrom, weekTo := GetSirusDates()
	url := fmt.Sprintf("https://sirus.su/api/base/x3/leaderboard/pve?ladder=players&type=%s&aggregation=max&week_from=%s&week_to=%s&ilvl_from=100&ilvl_to=300&page=%d&i=%d&boss=%d&specs=%d:%d",
		role, weekFrom, weekTo, page, raidID, bossID, classID, specID)

	var res Leaderboard
	if err := makeRequest(url, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func FetchLeaderboard(raidID, bossID, classID, specID int, role string) ([]LeaderboardPlayer, error) {
	var allPlayers []LeaderboardPlayer

	firstPage, err := GetPage(raidID, bossID, classID, specID, role, 1)
	if err != nil {
		time.Sleep(15 * time.Second)
		return nil, err
	}
	time.Sleep(2 * time.Second)
	allPlayers = append(allPlayers, firstPage.Data...)
	totalPages := firstPage.Meta.LastPage

	for p := 2; p <= totalPages; p++ {
		nextPage, err := GetPage(raidID, bossID, classID, specID, role, p)

		if err != nil {
			fmt.Printf("Error loading page %d: %v — skipping remaining pages\n", p, err)
			break
		}
		allPlayers = append(allPlayers, nextPage.Data...)
		time.Sleep(2 * time.Second)
	}

	return allPlayers, nil
}

func FetchMetasirusLeaderboard(mapID, bossID, difficulty int) ([]MetasirusLeaderboardPlayer, error) {
	var allPlayers []MetasirusLeaderboardPlayer

	fmt.Println("Парсинг первой страницы")
	firstPage, err := GetMetasirusLbPage(mapID, bossID, difficulty, 1)
	if err != nil {
		time.Sleep(15 * time.Second)
		return nil, err
	}
	time.Sleep(1 * time.Second)
	allPlayers = append(allPlayers, firstPage.Data...)
	totalPages := firstPage.Meta.LastPage

	for p := 2; p <= totalPages; p++ {
		fmt.Printf("Парсинг страинцы %d/%d\n", p, totalPages)
		nextPage, err := GetMetasirusLbPage(mapID, bossID, difficulty, p)
		if err != nil {
			log.Printf("GetMetasirusLbPage Error loading page %d: %v\n", p, err)
			time.Sleep(15 * time.Second)
			continue
		}

		allPlayers = append(allPlayers, nextPage.Data...)
		time.Sleep(1 * time.Second)
	}

	return allPlayers, nil
}

func GetMetasirusLbPage(mapID, bossID, difficulty, page int) (*MetasirusLeaderboard, error) {
	fmt.Printf("GetMetasirusLbPage  mapID:%d, bossID:%d, difficulty:%d page:%d\n", mapID, bossID, difficulty, page)
	url := fmt.Sprintf("https://metasirus.su/api/realm/22/map/%d/boss/%d/aggregation/character?difficulty=%d&type=&spec=&specs=&date=current&ilvl_from=&ilvl_to=&page=%d",
		mapID, bossID, difficulty, page)

	var res MetasirusLeaderboard
	if err := makeMetasirusRequest(url, &res); err != nil {
		return nil, err
	}
	fmt.Printf("Получен MetasirusLeaderboard от сервера, ДПС первого игрока %d\n", res.Data[0].Dps)
	return &res, nil
}

func FetchActualRaids() (ActualRaids, error) {
	url := "https://sirus.su/api/base/22/progression/pve/realm-progress"
	var res ActualRaids

	if err := makeRequest(url, &res); err != nil {
		return nil, err
	}

	return res, nil
}

func FetchGuildLatestBossKills(guildID int) (*LatestBossKills, error) {
	page := 1
	weekFrom, weekTo := GetLastWeek()
	url := fmt.Sprintf("https://sirus.su/api/base/22/progression/pve/latest-boss-kills?page=%d&week_from=%s&week_to=%s&guild=%d", page, weekFrom, weekTo, guildID)
	var res LatestBossKills

	if err := makeRequest(url, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func FetchPlayerLatestBossKills(playerID int) (*LatestPlayerBossKills, error) {
	page := 1
	weekFrom, weekTo := GetLastWeek()
	url := fmt.Sprintf("https://sirus.su/api/base/22/statistics/%d/pve/latest-boss-kills?page=%d&week_from=%s&week_to=%s", playerID, page, weekFrom, weekTo)
	var res LatestPlayerBossKills

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

func FetchPlayerID(name string) (int, error) {
	url := fmt.Sprintf("https://sirus.su/api/base/22/character/%s", url.QueryEscape(name))
	var res PlayerProfile

	err := makeRequest(url, &res)
	if err != nil {
		return 0, err
	}

	return res.Player.ID, nil
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
	lastThursday := now.AddDate(0, 0, -daysSinceThursday-7)

	weekFrom := lastThursday.Format("2006-01-02")

	weekTo := now.Format("2006-01-02")
	return weekFrom, weekTo
}

func GetLastWeek() (string, string) {
	loc := time.FixedZone("MSK", 3*3600)
	return calculateWeek(time.Now().In(loc))
}

func calculateWeek(now time.Time) (string, string) {
	daysSinceThursday := int(now.Weekday()) - int(time.Thursday)
	if daysSinceThursday < 0 {
		daysSinceThursday += 7
	}
	lastThursday := now.AddDate(0, 0, -daysSinceThursday)

	weekFrom := lastThursday.Format("2006-01-02")

	weekTo := now.Format("2006-01-02")
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
		fmt.Printf("makeRequest try #%d\n", try)
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("Attempt %d failed (network error): %v\n", try, err)
			time.Sleep(5 * time.Second)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("API returned status: %d\n", resp.StatusCode)
			resp.Body.Close()
			log.Printf("Attempt %d failed (status %d)\n", try, resp.StatusCode)
			time.Sleep(5 * time.Second)
			continue
		}
		defer resp.Body.Close()
		return json.NewDecoder(resp.Body).Decode(target)
	}

	return fmt.Errorf("all request attempts failed: %w", lastErr)
}

func makeMetasirusRequest(url string, target any) error {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoSandbox,
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.DisableGPU,
		chromedp.Flag("single-process", true),
		chromedp.Flag("headless", "new"),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disk-cache-size", "1048576"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	pageCtx, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()
	defer cancel()
	var body string
	err := chromedp.Run(pageCtx,
		chromedp.Navigate(url),
		chromedp.Sleep(10*time.Second),
		chromedp.Text(`body`, &body, chromedp.ByQuery),
	)

	if err != nil {
		return fmt.Errorf("chromedp error: %w", err)
	}

	body = strings.TrimSpace(body)

	return json.Unmarshal([]byte(body), target)
}
