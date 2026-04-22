package discord

type PlayerReport struct {
	Name     string
	Dps      int
	Hps      int
	Ilvl     int
	SpecName string
	SpecRank int
	Role     int
	ClassID  int
}

type BossKillReport struct {
	MapName   string
	BossName  string
	RaidOrder int
	KillID    int
	Duration  string
	Attempts  int
	TotalDps  int
	TotalHps  int
	KilledAt  string
	GuildID   int
	GuildName string
	Players   []PlayerReport
}
