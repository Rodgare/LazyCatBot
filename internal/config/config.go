package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DiscordToken   string
	DebugChannelID string
	IsDebug        bool
	SirusBaseURLs  []string
	DefaultRealm   string
	MythicSeason   int
	MythicWeekID   int
}

func LoadConfig() *Config {
	debugStr := os.Getenv("DEBUG")
	isDebug := debugStr == "true"

	sirusURLsStr := os.Getenv("SIRUS_BASE_URLS")
	var sirusURLs []string
	if sirusURLsStr != "" {
		sirusURLs = strings.Split(sirusURLsStr, ",")
	} else {
		sirusURLs = []string{"https://sirus.su"}
	}

	cleanedURLs := cleanURLs(sirusURLs)

	return &Config{
		DiscordToken:   os.Getenv("DISCORD_TOKEN"),
		DebugChannelID: os.Getenv("DEBUG_CHANNEL_ID"),
		IsDebug:        isDebug,
		SirusBaseURLs:  cleanedURLs,
		DefaultRealm:   getDefaultRealm(),
		MythicSeason:   getIntEnv("SIRUS_MYTHIC_SEASON", 6),
		MythicWeekID:   getIntEnv("SIRUS_MYTHIC_WEEK_ID", 0),
	}
}

func getIntEnv(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		return def
	}
	return n
}

func getDefaultRealm() string {
	realm := os.Getenv("SIRUS_REALM")
	if realm == "" {
		return "x3"
	}
	return realm
}

func (c *Config) GetSirusURLs() []string {
	return c.SirusBaseURLs
}

func cleanURLs(urls []string) []string {
	var cleaned []string
	seen := make(map[string]bool)
	for _, u := range urls {
		trimmed := strings.TrimRight(strings.TrimSpace(u), "/")
		if trimmed != "" && !seen[trimmed] {
			cleaned = append(cleaned, trimmed)
			seen[trimmed] = true
		}
	}
	return cleaned
}
