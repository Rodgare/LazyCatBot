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
	BossName string
	Duration string
	Attempts int
	TotalDps int
	Players  []PlayerReport
}
