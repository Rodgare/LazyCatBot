package models

import (
	"encoding/json"
	"fmt"
)

type PlayerLastAction struct {
	Type    string           `json:"type"`
	FightID json.RawMessage  `json:"id"`
	Date    string           `json:"datetime"`
}

type PlayerLastActions []PlayerLastAction

// FightIDValue coerces the boss fight id whether the API serializes it as a JSON
// number ("id": 9864425) or as a string ("id": "9864425"). Returns 0 when the
// field is missing or unparsable so a transient schema drift does not drop the
// whole kill-monitor fetch.
func (a *PlayerLastAction) FightIDValue() int {
	var id int
	if json.Unmarshal(a.FightID, &id) == nil {
		return id
	}
	var s string
	if json.Unmarshal(a.FightID, &s) == nil {
		var parsed int
		n, _ := fmt.Sscanf(s, "%d", &parsed)
		if n == 1 {
			return parsed
		}
	}
	return 0
}