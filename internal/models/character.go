package models

import "encoding/json"

type CharacterGem struct {
	Entry   int `json:"entry"`
	Quality int `json:"quality"`
}

type CharacterData struct {
	Titled    json.RawMessage `json:"titled"`
	Challenge struct {
		CurrentScore float64 `json:"current_score"`
	} `json:"challenge"`
	Equipments []struct {
		Gems []CharacterGem `json:"gems"`
	} `json:"equipments"`
}

type CharacterCache struct {
	Realm         string
	Name          string
	Title         string
	MythicRating  float64
	BlackDiamonds int
	UpdatedAt     int64
}

func isBlackDiamond(entry int) bool {
	if entry == 280505 {
		return true
	}
	if entry >= 103501 && entry <= 103520 {
		return true
	}
	if entry >= 100700 && entry <= 100885 {
		return true
	}
	return false
}

func (c *CharacterData) Title() string {
	if len(c.Titled) > 0 && c.Titled[0] == '"' {
		var s string
		if json.Unmarshal(c.Titled, &s) == nil {
			return s
		}
	}
	return ""
}

func (c *CharacterData) CountBlackDiamonds() int {
	count := 0
	for _, eq := range c.Equipments {
		for _, g := range eq.Gems {
			if isBlackDiamond(g.Entry) {
				count++
			}
		}
	}
	return count
}