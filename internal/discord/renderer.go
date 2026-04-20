package discord

import (
	"fmt"
	"slices"
	"strings"
)

// classANSI — ANSI-цвета для Discord ```ansi``` блока.
// Discord поддерживает ТОЛЬКО стандартные ANSI 30-37 (90-97 не работают).
// Подбираем ближайший цвет из 8 доступных.
var classANSI = map[int]string{
	1:  "\033[33m", // Warrior    → желтый (коричневого нет)
	2:  "\033[35m", // Paladin    → розовый/пурпурный
	3:  "\033[32m", // Hunter     → зеленый
	4:  "\033[33m", // Rogue      → желтый
	5:  "\033[37m", // Priest     → белый
	6:  "\033[31m", // DK         → красный
	7:  "\033[34m", // Shaman     → синий
	8:  "\033[36m", // Mage       → голубой/циан
	9:  "\033[35m", // Warlock    → фиолетовый/пурпурный
	11: "\033[31m", // Druid      → красный (оранжевого нет)
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
		rank := fmt.Sprintf("\033[30m%2d.\033[0m", i+1)
		name := ansi + truncate(p.Name, 12) + ansiReset
		spec := "\033[30m" + truncate(p.SpecName, 12) + ansiReset
		ilvl := fmt.Sprintf("\033[37m%3divl%s", p.Ilvl, ansiReset)
		perf := fmt.Sprintf("\033[37m%6s %s%s", formatNum(val), valLabel, ansiReset)
		srank := fmt.Sprintf("\033[33m#%-3d%s", p.SpecRank, ansiReset)

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
