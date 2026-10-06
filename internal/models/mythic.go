package models

type ReportMember struct {
	MemberGUID int    `json:"memberGuid"`
	Name       string `json:"name"`
	ClassID    int    `json:"classId"`
	SpecID     int    `json:"specId"`
	RoleID     int    `json:"roleId"`
	RaceID     int    `json:"raceId"`
	GenderID   int    `json:"genderId"`
	Ilvl       int    `json:"ilvl"`
	Zodiac     int    `json:"zodiac"`
}

type MythicRunItem struct {
	ID             int            `json:"id"`
	ChallengeID    int            `json:"challengeId"`
	ChallengeLevel int            `json:"challengeLevel"`
	Affixes        []int          `json:"affixes"`
	MapID          int            `json:"mapId"`
	Timer          int            `json:"timer"`
	RewardLevel    int            `json:"rewardLevel"`
	DeathCounter   int            `json:"deathCounter"`
	Score          float64        `json:"score"`
	DateTime       int            `json:"dateTime"`
	SeasonID       int            `json:"seasonId"`
	Template       int            `json:"template"`
	HasRunLog      bool           `json:"hasRunLog"`
	Members        []ReportMember `json:"members"`
	Position       int            `json:"position"`
}

type MythicRuns struct {
	Data []MythicRunItem `json:"data"`
}

type MythicMemberCombat struct {
	GUID        int `json:"guid"`
	DamageDone  int `json:"damageDone"`
	HealDone    int `json:"healDone"`
	DamageTaken int `json:"damageTaken"`
	Absorbed    int `json:"absorbed"`
	Deaths      int `json:"deaths"`
	Interrupts  int `json:"interrupts"`
}

type MythicRun struct {
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
	KeystoneLevel int            `json:"keystoneLevel"`
	SeasonID      int            `json:"seasonId"`
	WeekID        int            `json:"weekId"`
	TimeStart     int            `json:"timeStart"`
	DurationMs    int            `json:"durationMs"`
	Completed     bool           `json:"completed"`
	Version       int            `json:"version"`
	ForcesTotal   float64        `json:"forcesTotal"`
	ForcesMax     int            `json:"forcesMax"`
	ForcesPercent float64        `json:"forcesPercent"`
	Timer         int            `json:"timer"`
	RewardLevel   int            `json:"rewardLevel"`
	DeathCounter  int            `json:"deathCounter"`
	Score         float64        `json:"score"`
	Affixes       []int          `json:"affixes"`
	Members       []ReportMember `json:"members"`
	Frames        []struct {
		Players []MythicMemberCombat `json:"players"`
	} `json:"frames"`
}

type MythicReport struct {
	ID            int
	ChallengeID   int
	Name          string
	Icon          string
	MapID         int
	KeystoneLevel int
	SeasonID      int
	WeekID        int
	TimeStart     int
	DurationMs    int
	Completed     bool
	Version       int
	ForcesTotal   float64
	ForcesMax     int
	ForcesPercent float64
	Timer         int
	RewardLevel   int
	DeathCounter  int
	Score         float64
	Affixes       []int
	Members       []ReportMember
	DateTime      int
	Template      int
	HasRunLog     bool
	Realm         string
	CombatStats   []MythicMemberCombat
	HasCombat     bool
}

func (r *MythicRun) ToReport() MythicReport {
	return MythicReport{
		ID:            r.ID,
		ChallengeID:   r.ChallengeID,
		Name:          r.Name,
		Icon:          r.Icon,
		MapID:         r.MapID,
		KeystoneLevel: r.KeystoneLevel,
		SeasonID:      r.SeasonID,
		WeekID:        r.WeekID,
		TimeStart:     r.TimeStart,
		DurationMs:    r.DurationMs,
		Completed:     r.Completed,
		Version:       r.Version,
		ForcesTotal:   r.ForcesTotal,
		ForcesMax:     r.ForcesMax,
		ForcesPercent: r.ForcesPercent,
		Timer:         r.Timer,
		RewardLevel:   r.RewardLevel,
		DeathCounter:  r.DeathCounter,
		Score:         r.Score,
		Affixes:       r.Affixes,
		Members:       r.Members,
	}
}

func (item *MythicRunItem) ToReport() MythicReport {
	return MythicReport{
		ID:            item.ID,
		ChallengeID:   item.ChallengeID,
		KeystoneLevel: item.ChallengeLevel,
		Affixes:       item.Affixes,
		MapID:         item.MapID,
		Timer:         item.Timer,
		RewardLevel:   item.RewardLevel,
		DeathCounter:  item.DeathCounter,
		Score:         item.Score,
		DateTime:      item.DateTime,
		SeasonID:      item.SeasonID,
		Template:      item.Template,
		HasRunLog:     item.HasRunLog,
		Members:       item.Members,
	}
}

func (runs *MythicRuns) ToReports() []MythicReport {
	reports := make([]MythicReport, len(runs.Data))
	for i := range runs.Data {
		reports[i] = runs.Data[i].ToReport()
	}
	return reports
}

func (r *MythicReport) Enrich(details *MythicRun) {
	r.Name = details.Name
	r.Icon = details.Icon
	r.WeekID = details.WeekID
	r.TimeStart = details.TimeStart
	r.DurationMs = details.DurationMs
	r.Completed = details.Completed
	r.Version = details.Version
	r.ForcesTotal = details.ForcesTotal
	r.ForcesMax = details.ForcesMax
	r.ForcesPercent = details.ForcesPercent

	if len(details.Members) > 0 {
		r.Members = details.Members
	}
}

func (r *MythicReport) HasCombatDetails() bool {
	return r.HasRunLog && r.Completed && r.Timer > 0 && r.KeystoneLevel >= 10
}

func (r *MythicReport) ApplyCombat(details *MythicRun) {
	if details == nil || len(details.Frames) == 0 {
		return
	}

	aggregated := make(map[int]*MythicMemberCombat)
	for _, frame := range details.Frames {
		for i := range frame.Players {
			p := frame.Players[i]
			agg, ok := aggregated[p.GUID]
			if !ok {
				agg = &MythicMemberCombat{GUID: p.GUID}
				aggregated[p.GUID] = agg
			}
			agg.DamageDone += p.DamageDone
			agg.HealDone += p.HealDone
			agg.DamageTaken += p.DamageTaken
			agg.Absorbed += p.Absorbed
			agg.Deaths += p.Deaths
			agg.Interrupts += p.Interrupts
		}
	}

	duration := float64(details.DurationMs) / 1000
	for _, agg := range aggregated {
		if duration > 0 {
			agg.DamageDone = int(float64(agg.DamageDone) / duration)
			agg.HealDone = int(float64(agg.HealDone) / duration)
		}
		r.CombatStats = append(r.CombatStats, *agg)
	}
	r.HasCombat = true
}

type Affixes struct {
	Emoji string
	Name  string
	ID    int
}

var AffixMap = map[int]Affixes{
	382469: {Emoji: "<:382469:1555431378948988998>", Name: "Укрепленный", ID: 382469},
	382472: {Emoji: "<:382472:1555431399539081358>", Name: "Некротический", ID: 382472},
	382494: {Emoji: "<:382494:1555431398565744640>", Name: "Бушующий", ID: 382494},
	382491: {Emoji: "<:382491:1555431396489699468>", Name: "Злопамятный", ID: 382491},
	382468: {Emoji: "<:382468:1555431394933477417>", Name: "Тиранический", ID: 382468},
	382482: {Emoji: "<:382482:1555431393654349854>", Name: "Кровавый", ID: 382482},
	382470: {Emoji: "<:382470:1555431392450580480>", Name: "Взрывной", ID: 382470},
	382498: {Emoji: "<:382498:1555431391288758312>", Name: "Взрывоопасный", ID: 382498},
	382478: {Emoji: "<:382478:1555431389661372488>", Name: "Оплетающий", ID: 382478},
	382476: {Emoji: "<:382476:1555431388659060766>", Name: "Усиливающий", ID: 382476},
	382487: {Emoji: "<:382487:1555431387329208382>", Name: "Вулканический", ID: 382487},
	382505: {Emoji: "<:382505:1555431385668386837>", Name: "Мучительный", ID: 382505},
	382474: {Emoji: "<:382474:1555431383755653191>", Name: "Упрямый", ID: 382474},
	382485: {Emoji: "<:382485:1555431382074007552>", Name: "Разъяренный", ID: 382485},
	382502: {Emoji: "<:382502:1555431380719116388>", Name: "Сотрясающий", ID: 382502},
}
