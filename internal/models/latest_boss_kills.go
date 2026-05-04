package models

type LatestBossKills struct {
	Data []BossKill `json:"data"`
}

type BossKill struct {
	KillID  int    `json:"id"`
	GuildId int    `json:"guildId"`
	TimeEnd string `json:"timeEnd"`
}
