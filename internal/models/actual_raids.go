package models

type ActualSirusRaid struct {
	Order      int    `json:"order"`
	MapID      int    `json:"map_id"`
	MapName    string `json:"map_name"`
	Difficulty int    `json:"difficulty"`
	Background string `json:"background"`
	Equipment  int    `json:"equipment"`
	Actual     bool   `json:"actual"`
	Encounters []struct {
		Name string `json:"name"`
	} `json:"encounters"`
}

type ActualSirusRaids []ActualSirusRaid

type ActualRaid struct {
	RaidID int
	BossID int
}
