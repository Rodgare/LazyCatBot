package sirus

type RaidInfo struct {
	MapID      int    `json:"map_id"`
	MapName    string `json:"map_name"`
	Difficulty int    `json:"difficulty"`
	Actual     bool   `json:"actual"`
}

type CharacterResponse struct {
	Pve map[string]RaidInfo `json:"pve"`
}