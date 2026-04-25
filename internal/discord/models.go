package discord

type PlayerReport struct {
	Name            string
	Dps             int
	Hps             int
	Ilvl            int
	SpecID          int
	SpecName        string
	SpecRank        int
	SpecPercentile  int
	IlvlRank        int
	IlvlPercentile  int
	ClassRank       int
	ClassPercentile int
	OverallRank     int
	OverallPercentile int
	Role            int
	ClassID         int
	T4              int
	Zodiac          int
	Category        int
}

type LootReport struct {
	ID    int
	Name  string
	Count int
	Icon  string
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
	Loots     []LootReport
	Players   []PlayerReport
}
