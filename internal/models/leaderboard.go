package models

type Leaderboard struct {
	Data []LeaderboardPlayer `json:"data"`
	Meta struct {
		LastPage int `json:"last_page"`
	} `json:"meta"`
}

type LeaderboardPlayer struct {
	Name     string `json:"name"`
	Rank     int    `json:"rank"`
	Dps      int    `json:"dps"`
	Hps      int    `json:"hps"`
	ClassID  int    `json:"class_id"`
	SpecID   int    `json:"spec"`
	Ilvl     int    `json:"ilvl"`
	GuildID  int    `json:"guild_id"`
	Category int    `json:"category"`
	Zodiac   struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"zodiac"`
	Itemset [][]int `json:"itemset"`
}
