package sirus

type Leaderboard struct {
	Data []LeaderboardPlayer `json:"data"`
	Meta struct {
		LastPage int `json:"last_page"`
	} `json:"meta"`
}

type LeaderboardPlayer struct {
	Name    string `json:"name"`
	Rank    int    `json:"rank"`
	Dps     int    `json:"dps"`
	ClassID int    `json:"class_id"`
	SpecID  int    `json:"spec"`
	Ilvl    int    `json:"ilvl"`
}

type LatestBossKills struct {
	Data []BossKill `json:"data"`
}

type BossKill struct {
	KillID  int    `json:"id"`
	GuildId int    `json:"guildId"`
	TimeEnd string `json:"timeEnd"`
}

type LatestPlayerBossKills struct {
	Data []PlayerBossKill `json:"data"`
}

type PlayerBossKill struct {
	ID int `json:"id"`
}

type Itemset struct {
	Id    int    `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type Player struct {
	GUID     int    `json:"guid"`
	Name     string `json:"name"`
	Level    int    `json:"level"`
	ClassID  int    `json:"class_id"`
	RaceID   int    `json:"race_id"`
	Gender   int    `json:"gender"`
	Spec     int    `json:"spec"`
	Ilvl     int    `json:"ilvl"`
	Category int    `json:"category"`
	Dps      int    `json:"dps"`
	Hps      int    `json:"hps"`
	Guild    struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Level int    `json:"level"`
	} `json:"guild"`
	Zodiac struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"zodiac"`
	Itemset []Itemset `json:"itemset"`
}

type BossFight struct {
	Data struct {
		BossName string `json:"boss_name"`
		MapName  string `json:"map_name"`
		Loots    []struct {
			Entry int `json:"entry"`
			Count int `json:"count"`
			Item  struct {
				Entry   int    `json:"entry"`
				Quality int    `json:"quality"`
				Icon    string `json:"icon"`
				Name    string `json:"name"`
				Color   string `json:"color"`
				RealmID int    `json:"realm_id"`
			} `json:"item"`
		} `json:"loots"`
		Attempts   int `json:"attempts"`
		Difficulty int `json:"difficulty"`
		Guild      struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"guild"`
		KilledAt    string   `json:"killed_at"`
		FightLength string   `json:"fight_length"`
		Players     []Player `json:"players"`
	} `json:"data"`
	Order     int `json:"order"`
	Encounter int `json:"encounter"`
}

type ActualRaid struct {
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

type ActualRaids []ActualRaid

type PlayerProfile struct {
	Player struct {
		ID int `json:"guid"`
	} `json:"character"`
}
