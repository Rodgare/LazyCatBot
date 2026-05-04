package models

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

type LootReport struct {
	ID    int
	Name  string
	Count int
	Icon  string
}
