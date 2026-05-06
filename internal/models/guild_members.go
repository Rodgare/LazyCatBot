package models

type Guild struct {
	Guild struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Level int    `json:"level"`
	} `json:"guild"`
	Ranks []struct {
		Rid   int    `json:"rid"`
		Rname string `json:"rname"`
	} `json:"ranks"`
	Members             []GuildMembers `json:"members"`
	GuildFinderSettings struct {
		Availability int    `json:"availability"`
		ClassRoles   int    `json:"classRoles"`
		Interests    int    `json:"interests"`
		Level        int    `json:"level"`
		ActiveTime   int    `json:"activeTime"`
		Comment      string `json:"comment"`
	} `json:"guildFinderSettings"`
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
