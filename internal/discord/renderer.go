package discord

import (
	"fmt"
	"slices"
	"strings"
)

// classANSI — ANSI-цвета для Discord ```ansi``` блока.
// Discord поддерживает стандартные ANSI 30-37 и яркие 90-97.
// Подбираем ближайший к реальному цвету класса WoW.
var classANSI = map[int]string{
	1:  "\033[33m",  // Warrior    — #C79C6E → жёлтый
	2:  "\033[95m",  // Paladin    — #F58CBA → ярко-пурпурный
	3:  "\033[32m",  // Hunter     — #ABD473 → зелёный
	4:  "\033[93m",  // Rogue      — #FFF569 → ярко-жёлтый
	5:  "\033[97m",  // Priest     — #FFFFFF → белый
	6:  "\033[31m",  // DK         — #C41F3B → красный
	7:  "\033[34m",  // Shaman     — #0070DE → синий
	8:  "\033[96m",  // Mage       — #69CCF0 → голубой
	9:  "\033[35m",  // Warlock    — #9482C9 → пурпурный
	11: "\033[91m",  // Druid      — #FF7D0A → ярко-красный (оранжевого нет в ANSI)
}

const ansiReset = "\033[0m"

// BuildReportText строит текстовый отчёт для Discord embed.
// Возвращает заголовок раздела и ANSI-раскрашенный code-блок.
func BuildReportText(report BossKillReport) (ddBlock string, healBlock string) {
	var dds, healers []PlayerReport
	for _, p := range report.Players {
		if p.Role == 1 {
			healers = append(healers, p)
		} else {
			dds = append(dds, p)
		}
	}

	slices.SortFunc(dds, func(a, b PlayerReport) int { return b.Dps - a.Dps })
	slices.SortFunc(healers, func(a, b PlayerReport) int { return b.Hps - a.Hps })

	ddBlock = buildAnsiBlock(dds, true)
	healBlock = buildAnsiBlock(healers, false)
	return
}

func buildAnsiBlock(players []PlayerReport, isDD bool) string {
	if len(players) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("```ansi\n")

	for i, p := range players {
		ansi := classANSI[p.ClassID]
		if ansi == "" {
			ansi = "\033[37m"
		}

		val := p.Dps
		valLabel := "DPS"
		if !isDD {
			val = p.Hps
			valLabel = "HPS"
		}

		// Форматируем строку: ранг, имя (цвет класса), спек, iLvl, DPS/HPS, ранг по спеку
		rank := fmt.Sprintf("\033[90m%2d.\033[0m", i+1)
		name := ansi + truncate(p.Name, 12) + ansiReset
		spec := "\033[90m" + truncate(p.SpecName, 12) + ansiReset
		ilvl := fmt.Sprintf("\033[37m%3divl%s", p.Ilvl, ansiReset)
		perf := fmt.Sprintf("\033[97m%6s %s%s", formatNum(val), valLabel, ansiReset)
		srank := fmt.Sprintf("\033[93m#%-3d%s", p.SpecRank, ansiReset)

		sb.WriteString(fmt.Sprintf("%s %s  %s  %s  %s  %s\n",
			rank, name, spec, ilvl, perf, srank))
	}

	sb.WriteString("```")
	return sb.String()
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		// Дополняем пробелами для выравнивания
		return s + strings.Repeat(" ", max-len(runes))
	}
	return string(runes[:max-1]) + "…"
}

func formatNum(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}
