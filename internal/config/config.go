package config

import (
	"encoding/json"
	"os"
	"strings"
)

type Config struct {
	SirusBaseURL  string   `json:"sirus_base_url"`
	SirusBaseURLs []string `json:"sirus_base_urls"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return defaultConfig(), nil
	}
	defer file.Close()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return defaultConfig(), nil
	}

	return &cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		SirusBaseURLs: []string{"https://sirus.su", "https://sirus.org"},
	}
}

func (c *Config) GetSirusURLs() []string {
	var urls []string
	if len(c.SirusBaseURLs) > 0 {
		urls = append(urls, c.SirusBaseURLs...)
	}
	if c.SirusBaseURL != "" {
		urls = append(urls, c.SirusBaseURL)
	}

	if len(urls) == 0 {
		urls = []string{"https://sirus.su", "https://sirus.org"}
	}

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
