package models

type Guild struct {
	Members []GuildMembers `json:"members"`
}

type GuildMembers struct {
	GUID   int    `json:"guid"`
	Name   string `json:"name"`
	Race   int    `json:"race"`
	Class  int    `json:"class"`
	Level  int    `json:"level"`
	Gender int    `json:"gender"`
	Ilvl   int    `json:"ilvl"`
	Skills []struct {
		Skill int `json:"skill"`
		Value int `json:"value"`
		Max   int `json:"max"`
	} `json:"skills"`
	Rank    int `json:"rank"`
	Faction int `json:"faction"`
}
