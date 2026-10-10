package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DiscordToken    string
	DebugChannelID  string
	IsDebug         bool
	SirusBaseURLs   []string
	DefaultRealm    string
	Realms          []string
	SecondaryRealms []string
	MythicSeason    int
	MythicWeekID    int
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
		DiscordToken:    os.Getenv("DISCORD_TOKEN"),
		DebugChannelID:  os.Getenv("DEBUG_CHANNEL_ID"),
		IsDebug:         isDebug,
		SirusBaseURLs:   cleanedURLs,
		DefaultRealm:    getDefaultRealm(),
		Realms:          getStringListEnv("SIRUS_REALMS", []string{"x3", "x5"}),
		SecondaryRealms: getStringListEnv("SIRUS_SECONDARY_REALMS", []string{"x5"}),
		MythicSeason:    getIntEnv("SIRUS_MYTHIC_SEASON", 6),
		MythicWeekID:    getIntEnv("SIRUS_MYTHIC_WEEK_ID", 0),
	}
}

func getStringListEnv(key string, def []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	var parts []string
	for _, s := range strings.Split(v, ",") {
		t := strings.TrimSpace(s)
		if t != "" {
			parts = append(parts, t)
		}
	}
	return parts
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

// IsSecondaryRealm reports whether realm is a secondary server that must only
// parse actual (current) raids instead of the full history.
func (c *Config) IsSecondaryRealm(realm string) bool {
	for _, r := range c.SecondaryRealms {
		if r == realm {
			return true
		}
	}
	return false
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
