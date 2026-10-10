package sirus

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)

var setTier = make(map[int]int)
var setTierLoaded = false

type itemsetInfo struct {
	SetID    int    `json:"setId"`
	TierName string `json:"tierName"`
}

type itemsetsDoc struct {
	Sets []itemsetInfo `json:"sets"`
}

func loadSetTiers() {
	if setTierLoaded {
		return
	}
	setTierLoaded = true

	data, err := os.ReadFile("itemsets.json")
	if err != nil {
		slog.Error("Cannot read itemsets.json", "path", "itemsets.json", "error", err)
		return
	}

	var doc itemsetsDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		slog.Error("Failed to parse itemsets.json", "error", err)
		return
	}

	for _, set := range doc.Sets {
		if tier := parseTierName(set.TierName); tier > 0 {
			setTier[set.SetID] = tier
		}
	}
}

func parseTierName(name string) int {
	var tier int
	if _, err := fmt.Sscanf(name, "Т%d", &tier); err != nil {
		return 0
	}
	return tier
}

func GetSetTierForID(setID int) int {
	loadSetTiers()
	if tier, ok := setTier[setID]; ok {
		return tier
	}
	return 0
}