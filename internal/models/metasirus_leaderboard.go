package models

import "time"

type MetasirusLeaderboard struct {
	Data  []MetasirusLeaderboardPlayer `json:"data"`
	Links struct {
		First string `json:"first"`
		Last  string `json:"last"`
		Prev  string `json:"prev"`
		Next  string `json:"next"`
	} `json:"links"`
	Meta struct {
		CurrentPage int `json:"current_page"`
		From        int `json:"from"`
		LastPage    int `json:"last_page"`
		Links       []struct {
			URL    string `json:"url"`
			Label  string `json:"label"`
			Page   int    `json:"page"`
			Active bool   `json:"active"`
		} `json:"links"`
		Path    string `json:"path"`
		PerPage int    `json:"per_page"`
		To      int    `json:"to"`
		Total   int    `json:"total"`
	} `json:"meta"`
}

type MetasirusLeaderboardPlayer struct {
	Difficulty      string    `json:"difficulty"`
	Ilvl            int       `json:"ilvl"`
	Hps             int       `json:"hps"`
	Dps             int       `json:"dps"`
	KilledAt        time.Time `json:"killed_at"`
	FightLength     int       `json:"fight_length"`
	FightID         int       `json:"fight_id"`
	BossID          int       `json:"boss_id"`
	MapID           int       `json:"map_id"`
	CategoryID      int       `json:"category_id"`
	CharacterGUID   int       `json:"character_guid"`
	MetasirusSpecID int       `json:"spec_id"`
	ClassID         int       `json:"type_id"`
	Percentile      int       `json:"percentile"`
	ZodiacID        int       `json:"zodiac_id"`
	Sets            string    `json:"sets"`
	Character       struct {
		Name       string `json:"name"`
		RealmID    int    `json:"realm_id"`
		GUID       int    `json:"guid"`
		Ilvl       any    `json:"ilvl"`
		Gender     int    `json:"gender"`
		ClassID    int    `json:"type_id"`
		SpecID     any    `json:"spec_id"`
		ID         int    `json:"id"`
		Title      any    `json:"title"`
		Frame      any    `json:"frame"`
		Background any    `json:"background"`
		Guild      struct {
			Name    string    `json:"name"`
			RealmID int       `json:"realm_id"`
			GUID    int       `json:"guid"`
			Faction any       `json:"faction"`
			RealmCd time.Time `json:"realm_cd"`
			APIURL  string    `json:"api_url"`
		} `json:"guild"`
	} `json:"character"`
}
