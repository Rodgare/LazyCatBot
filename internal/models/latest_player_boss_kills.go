package models

type LatestPlayerBossKills struct {
	Data []PlayerBossKill `json:"data"`
}

type PlayerBossKill struct {
	ID int `json:"id"`
}
