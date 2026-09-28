package models

type MythicPlus struct {
	Data []struct {
		ID             int   `json:"id"`
		ChallengeID    int   `json:"challengeId"`
		ChallengeLevel int   `json:"challengeLevel"`
		Affixes        []int `json:"affixes"`
		MapID          int   `json:"mapId"`
		Timer          int   `json:"timer"`
		RewardLevel    int   `json:"rewardLevel"`
		DeathCounter   int   `json:"deathCounter"`
		Score          int   `json:"score"`
		DateTime       int   `json:"dateTime"`
		SeasonID       int   `json:"seasonId"`
		Template       int   `json:"template"`
		HasRunLog      bool  `json:"hasRunLog"`
		Members        []struct {
			MemberGUID int    `json:"memberGuid"`
			Name       string `json:"name"`
			ClassID    int    `json:"classId"`
			SpecID     int    `json:"specId"`
			RoleID     int    `json:"roleId"`
			RaceID     int    `json:"raceId"`
			GenderID   int    `json:"genderId"`
		} `json:"members"`
		Position int `json:"position"`
	} `json:"data"`
	Challenges struct {
		Num4 struct {
			ChallengeID int    `json:"challengeId"`
			Name        string `json:"name"`
			Icon        string `json:"icon"`
		} `json:"4"`
		Num5 struct {
			ChallengeID int    `json:"challengeId"`
			Name        string `json:"name"`
			Icon        string `json:"icon"`
		} `json:"5"`
		Num6 struct {
			ChallengeID int    `json:"challengeId"`
			Name        string `json:"name"`
			Icon        string `json:"icon"`
		} `json:"6"`
		Num8 struct {
			ChallengeID int    `json:"challengeId"`
			Name        string `json:"name"`
			Icon        string `json:"icon"`
		} `json:"8"`
		Num9 struct {
			ChallengeID int    `json:"challengeId"`
			Name        string `json:"name"`
			Icon        string `json:"icon"`
		} `json:"9"`
		Num10 struct {
			ChallengeID int    `json:"challengeId"`
			Name        string `json:"name"`
			Icon        string `json:"icon"`
		} `json:"10"`
		Num11 struct {
			ChallengeID int    `json:"challengeId"`
			Name        string `json:"name"`
			Icon        string `json:"icon"`
		} `json:"11"`
		Num13 struct {
			ChallengeID int    `json:"challengeId"`
			Name        string `json:"name"`
			Icon        string `json:"icon"`
		} `json:"13"`
	} `json:"challenges"`
	Meta struct {
		CurrentPage int `json:"current_page"`
		LastPage    int `json:"last_page"`
		PerPage     int `json:"per_page"`
		Total       int `json:"total"`
	} `json:"meta"`
}

type Runlog struct {
	ID          int    `json:"id"`
	ChallengeID int    `json:"challengeId"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	MapID       int    `json:"mapId"`
	Map         struct {
		AreaID       int   `json:"areaId"`
		Floors       []int `json:"floors"`
		DefaultFloor int   `json:"defaultFloor"`
	} `json:"map"`
	Positions []struct {
		GUID    int `json:"guid"`
		Samples []struct {
			T     int     `json:"t"`
			X     float64 `json:"x"`
			Y     float64 `json:"y"`
			Z     float64 `json:"z"`
			Mx    float64 `json:"mx"`
			My    float64 `json:"my"`
			Floor int     `json:"floor"`
		} `json:"samples"`
	} `json:"positions"`
	PositionInterval int     `json:"positionInterval"`
	KeystoneLevel    int     `json:"keystoneLevel"`
	SeasonID         int     `json:"seasonId"`
	WeekID           int     `json:"weekId"`
	TimeStart        int     `json:"timeStart"`
	DurationMs       int     `json:"durationMs"`
	Completed        bool    `json:"completed"`
	Version          int     `json:"version"`
	ForcesTotal      float64 `json:"forcesTotal"`
	ForcesMax        int     `json:"forcesMax"`
	ForcesPercent    float64 `json:"forcesPercent"`
	Timer            int     `json:"timer"`
	RewardLevel      int     `json:"rewardLevel"`
	DeathCounter     int     `json:"deathCounter"`
	Score            float64 `json:"score"`
	Affixes          []int   `json:"affixes"`
	Members          []struct {
		MemberGUID int    `json:"memberGuid"`
		Name       string `json:"name"`
		ClassID    int    `json:"classId"`
		SpecID     int    `json:"specId"`
		RoleID     int    `json:"roleId"`
		RaceID     int    `json:"raceId"`
		GenderID   int    `json:"genderId"`
		Ilvl       int    `json:"ilvl"`
		Zodiac     int    `json:"zodiac"`
	} `json:"members"`
	Frames []struct {
		ID        int     `json:"id"`
		TimeStart int     `json:"timeStart"`
		TimeEnd   int     `json:"timeEnd"`
		Duration  int     `json:"duration"`
		Boss      bool    `json:"boss"`
		Encounter any     `json:"encounter"`
		AreaID    int     `json:"areaId"`
		Forces    float64 `json:"forces"`
		Creatures []struct {
			GUID      int    `json:"guid"`
			Entry     int    `json:"entry"`
			Name      string `json:"name"`
			TimeEnter int    `json:"timeEnter"`
			Enter     struct {
				X     float64 `json:"x"`
				Y     float64 `json:"y"`
				Z     float64 `json:"z"`
				Mx    float64 `json:"mx"`
				My    float64 `json:"my"`
				Floor int     `json:"floor"`
			} `json:"enter"`
			TimeEnd int `json:"timeEnd"`
			End     struct {
				X     float64 `json:"x"`
				Y     float64 `json:"y"`
				Z     float64 `json:"z"`
				Mx    float64 `json:"mx"`
				My    float64 `json:"my"`
				Floor int     `json:"floor"`
			} `json:"end"`
			EndReason  int     `json:"endReason"`
			KillPoints float64 `json:"killPoints"`
		} `json:"creatures"`
		Players []struct {
			GUID        int `json:"guid"`
			DamageDone  int `json:"damageDone"`
			HealDone    int `json:"healDone"`
			DamageTaken int `json:"damageTaken"`
			Absorbed    int `json:"absorbed"`
			Deaths      int `json:"deaths"`
			Interrupts  int `json:"interrupts"`
		} `json:"players"`
		PlayerPositions []any `json:"playerPositions"`
		Deaths          []any `json:"deaths"`
		Interrupts      []any `json:"interrupts"`
		HasCombatlog    bool  `json:"hasCombatlog"`
	} `json:"frames"`
}
