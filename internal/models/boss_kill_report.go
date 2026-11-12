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
	Realm     string
	Loots     []LootReport
	Players   []PlayerReport
	// RangeLabel is the human-readable ranking window shown on rendered
	// ranking images (e.g. "4 КД"). Empty means the default "2 кд".
	RangeLabel string
}

type LootReport struct {
	ID    int
	Name  string
	Count int
	Icon  string
}
