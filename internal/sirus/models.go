package sirus

type LeaderboardPlayer struct {
	Name string `json:"name"`
	Rank int    `json:"rank"`
	Dps  int    `json:"dps"`
}

type Leaderboard struct {
	Data []LeaderboardPlayer `json:"data"`
	Meta struct {
		LastPage int `json:"last_page"`
	} `json:"meta"`
}
