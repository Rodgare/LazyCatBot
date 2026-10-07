package sirus

import (
	"LazyCatBot/internal/models"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	logger        *slog.Logger
	sirusBaseURLs []string
	httpClient    *http.Client
}

func NewClient(logger *slog.Logger, sirusBaseURLs []string) *Client {
	var cleaned []string
	for _, u := range sirusBaseURLs {
		trimmed := strings.TrimRight(strings.TrimSpace(u), "/")
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	if len(cleaned) == 0 {
		cleaned = []string{"https://sirus.su"}
	}

	return &Client{
		logger:        logger,
		sirusBaseURLs: cleaned,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}
}

func normalizeRealm(realm string) string {
	if realm == "" {
		return "x3"
	}
	return realm
}

func (c *Client) GetPage(realm string, raidID, bossID, classID, specID int, role string, page int) (*models.Leaderboard, error) {
	realm = normalizeRealm(realm)
	weekFrom, weekTo := c.GetSirusDates()
	endpoint := fmt.Sprintf("/api/base/%s/leaderboard/pve?ladder=players&type=%s&aggregation=max&week_from=%s&week_to=%s&ilvl_from=100&ilvl_to=300&page=%d&i=%d&boss=%d&specs=%d:%d",
		realm, role, weekFrom, weekTo, page, raidID, bossID, classID, specID)

	var res models.Leaderboard
	if err := c.makeRequest(endpoint, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *Client) FetchLeaderboard(realm string, raidID, bossID, classID, specID int, role string) ([]models.LeaderboardPlayer, error) {
	var allPlayers []models.LeaderboardPlayer

	firstPage, err := c.GetPage(realm, raidID, bossID, classID, specID, role, 1)
	if err != nil {
		time.Sleep(15 * time.Second)
		return nil, err
	}
	time.Sleep(2 * time.Second)
	allPlayers = append(allPlayers, firstPage.Data...)
	totalPages := firstPage.Meta.LastPage

	for p := 2; p <= totalPages; p++ {
		nextPage, err := c.GetPage(realm, raidID, bossID, classID, specID, role, p)

		if err != nil {
			c.logger.Error("Error fetch leaderboard", "error", err, "page", p)
			break
		}
		allPlayers = append(allPlayers, nextPage.Data...)
		time.Sleep(2 * time.Second)
	}

	return allPlayers, nil
}

func (c *Client) FetchActualRaids(realm string) (models.ActualSirusRaids, error) {
	realm = normalizeRealm(realm)
	endpoint := fmt.Sprintf("/api/base/%s/progression/pve/realm-progress", realm)
	var res models.ActualSirusRaids

	if err := c.makeRequest(endpoint, &res); err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Client) GetLatestMythicRuns(realm string) (*models.MythicRuns, error) {
	realm = normalizeRealm(realm)
	endpoint := fmt.Sprintf("/api/base/%s/leaderboard/challenge/latest-runs", realm)

	var res models.MythicRuns

	if err := c.makeRequest(endpoint, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *Client) FetchMythicRunDetails(realm string, runID int) (*models.MythicRun, error) {
	realm = normalizeRealm(realm)
	endpoint := fmt.Sprintf("/api/base/%s/details/challenge/%d/runlog", realm, runID)

	var res models.MythicRun

	if err := c.makeRequest(endpoint, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *Client) FetchGuildLatestBossKills(realm string, guildID int) (*models.LatestBossKills, error) {
	realm = normalizeRealm(realm)
	page := 1
	weekFrom, weekTo := c.GetLastWeek()
	endpoint := fmt.Sprintf("/api/base/%s/progression/pve/latest-boss-kills?page=%d&week_from=%s&week_to=%s&guild=%d", realm, page, weekFrom, weekTo, guildID)
	var res models.LatestBossKills

	if err := c.makeRequest(endpoint, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *Client) FetchPlayerLatestBossKills(realm string, playerID int) (*models.LatestPlayerBossKills, error) {
	realm = normalizeRealm(realm)
	page := 1
	weekFrom, weekTo := c.GetLastWeek()
	endpoint := fmt.Sprintf("/api/base/%s/statistics/%d/pve/latest-boss-kills?page=%d&week_from=%s&week_to=%s", realm, playerID, page, weekFrom, weekTo)
	var res models.LatestPlayerBossKills

	if err := c.makeRequest(endpoint, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *Client) FetchPlayerLastActions(realm string, pID int) (*models.PlayerLastActions, error) {
	realm = normalizeRealm(realm)
	endpoint := fmt.Sprintf("/api/base/%s/statistics/%d/latest-actions", realm, pID)

	var res models.PlayerLastActions

	if err := c.makeRequest(endpoint, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *Client) FetchBossFightDetails(realm string, fightID int) (*models.BossFight, error) {
	realm = normalizeRealm(realm)
	endpoint := fmt.Sprintf("/api/base/%s/details/bossfight/%v", realm, fightID)
	var fight models.BossFight

	if err := c.makeRequest(endpoint, &fight); err != nil {
		return nil, err
	}

	return &fight, nil
}

func (c *Client) FetchPlayerID(realm string, name string) (int, error) {
	realm = normalizeRealm(realm)
	endpoint := fmt.Sprintf("/api/base/%s/character/%s", realm, url.QueryEscape(name))
	var res models.PlayerProfile

	err := c.makeRequest(endpoint, &res)
	if err != nil {
		return 0, err
	}

	return res.Player.ID, nil
}

func (c *Client) GetSirusDates() (string, string) {
	loc := time.FixedZone("MSK", 3*3600)
	return c.calculateSirusDates(time.Now().In(loc))
}

func (c *Client) calculateSirusDates(now time.Time) (string, string) {
	daysSinceThursday := int(now.Weekday()) - int(time.Thursday)
	if daysSinceThursday < 0 {
		daysSinceThursday += 7
	}
	lastThursday := now.AddDate(0, 0, -daysSinceThursday-7)

	weekFrom := lastThursday.Format("2006-01-02")

	weekTo := now.Format("2006-01-02")
	return weekFrom, weekTo
}

func (c *Client) GetLastWeek() (string, string) {
	loc := time.FixedZone("MSK", 3*3600)
	return c.calculateWeek(time.Now().In(loc))
}

func (c *Client) calculateWeek(now time.Time) (string, string) {
	daysSinceThursday := int(now.Weekday()) - int(time.Thursday)
	if daysSinceThursday < 0 {
		daysSinceThursday += 7
	}
	lastThursday := now.AddDate(0, 0, -daysSinceThursday)

	weekFrom := lastThursday.Format("2006-01-02")

	weekTo := now.Format("2006-01-02")
	return weekFrom, weekTo
}

func (c *Client) makeRequest(endpoint string, target any) error {
	var lastErr error
	for _, baseURL := range c.sirusBaseURLs {
		fullURL := fmt.Sprintf("%s%s", baseURL, endpoint)
		req, err := http.NewRequest("GET", fullURL, nil)
		if err != nil {
			return err
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) LazyCatBot/1.0")
		req.Header.Set("Accept", "application/json")

		backoff := time.Second
		for try := 1; try <= 3; try++ {
			resp, err := c.httpClient.Do(req)
			if err != nil {
				lastErr = err
				c.logger.Error("Http request failed", "url", fullURL, "error", err, "attempt", try)
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
			if resp.StatusCode != http.StatusOK {
				lastErr = fmt.Errorf("API returned status: %d", resp.StatusCode)
				resp.Body.Close()
				c.logger.Error("Http request failed", "url", fullURL, "error", lastErr, "status_code", resp.StatusCode, "attempt", try)
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
			err = json.NewDecoder(resp.Body).Decode(target)
			resp.Body.Close()
			if err != nil {
				return err
			}
			return nil
		}
	}

	return fmt.Errorf("all request attempts across all base URLs failed: %w", lastErr)
}

func (c *Client) FetchGuildMembers(realm string, guild_id int) (*[]models.GuildMembers, string, error) {
	realm = normalizeRealm(realm)
	var gms models.Guild

	endpoint := fmt.Sprintf("/api/base/%s/guild/%d", realm, guild_id)
	if err := c.makeRequest(endpoint, &gms); err != nil {
		return nil, "", err
	}

	return &gms.Members, gms.Name, nil
}
