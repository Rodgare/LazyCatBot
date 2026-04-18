package discord

type PlayerReport struct {
	Name     string
	Dps      int
	SpecRank int
}

type BossKillReport struct {
	BossName string
	Players  []PlayerReport
}
