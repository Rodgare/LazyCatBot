package discord

import (
	"bytes"
	"fmt"
	"os"
	"slices"

	"github.com/fogleman/gg"
)

var classColors = map[int]string{
	1:  "#C79C6E", // Warrior
	2:  "#F58CBA", // Paladin
	3:  "#ABD473", // Hunter
	4:  "#FFF569", // Rogue
	5:  "#FFFFFF", // Priest
	6:  "#C41F3B", // DK
	7:  "#0070DE", // Shaman
	8:  "#69CCF0", // Mage
	9:  "#9482C9", // Warlock
	11: "#FF7D0A", // Druid
}

func RenderReportImage(report BossKillReport) ([]byte, error) {
	const (
		width     = 800
		rowHeight = 35
		headerH   = 50
		margin    = 20
	)

	// Разделяем игроков на роли
	var dds, healers []PlayerReport
	for _, p := range report.Players {
		if p.Role == 1 {
			healers = append(healers, p)
		} else {
			dds = append(dds, p)
		}
	}

	// Сортируем: ДД по ДПС, Хилов по ХПС
	slices.SortFunc(dds, func(a, b PlayerReport) int { return b.Dps - a.Dps })
	slices.SortFunc(healers, func(a, b PlayerReport) int { return b.Hps - a.Hps })

	// Считаем общую высоту: заголовок рейда + секция ДД + секция Хилов
	totalRows := len(dds) + len(healers)
	height := headerH + (totalRows+2)*rowHeight + margin*2 // +2 для заголовков ролей

	dc := gg.NewContext(width, int(height))

	// Фон
	dc.SetHexColor("#2c2f33") // Темно-серый Discord
	dc.Clear()

	// Попытка загрузить шрифт (Windows path)
	fontPath := "C:\\Windows\\Fonts\\arial.ttf"
	if _, err := os.Stat(fontPath); err == nil {
		if err := dc.LoadFontFace(fontPath, 18); err != nil {
			fmt.Printf("Error loading font: %v\n", err)
		}
	}

	y := float64(margin)

	// Заголовок Босса
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(fmt.Sprintf("⚔️ %s", report.BossName), width/2, y+20, 0.5, 0.5)
	y += headerH

	// Секция ДД
	if len(dds) > 0 {
		y = drawRoleHeader(dc, "DAMAGE DEALERS", y, width)
		for i, p := range dds {
			y = drawPlayerRow(dc, i+1, p, y, width, rowHeight, true)
		}
	}

	// Секция Хилов
	if len(healers) > 0 {
		y += 10
		y = drawRoleHeader(dc, "HEALERS", y, width)
		for i, p := range healers {
			y = drawPlayerRow(dc, i+1, p, y, width, rowHeight, false)
		}
	}

	buf := new(bytes.Buffer)
	err := dc.EncodePNG(buf)
	return buf.Bytes(), err
}

func drawRoleHeader(dc *gg.Context, title string, y float64, width int) float64 {
	dc.SetRGBA(0, 0, 0, 0.3)
	dc.DrawRectangle(0, y, float64(width), 30)
	dc.Fill()

	dc.SetRGB(0.7, 0.7, 0.7)
	dc.DrawString(title, 20, y+20)
	return y + 35
}

func drawPlayerRow(dc *gg.Context, rank int, p PlayerReport, y float64, width int, rowHeight float64, isDD bool) float64 {
	// Подложка (чередующийся фон)
	if rank%2 == 0 {
		dc.SetRGBA(1, 1, 1, 0.05)
		dc.DrawRectangle(0, y, float64(width), rowHeight)
		dc.Fill()
	}

	// №
	dc.SetRGB(0.5, 0.5, 0.5)
	dc.DrawString(fmt.Sprintf("%d", rank), 20, y+25)

	// Имя (цвет класса)
	colorHex, ok := classColors[p.ClassID]
	if !ok {
		colorHex = "#FFFFFF"
	}
	dc.SetHexColor(colorHex)
	dc.DrawString(p.Name, 60, y+25)

	// Спек
	dc.SetRGB(0.6, 0.6, 0.6)
	dc.DrawString(p.SpecName, 220, y+25)

	// iLvl
	dc.SetRGB(0.8, 0.8, 0.8)
	dc.DrawString(fmt.Sprintf("%d", p.Ilvl), 380, y+25)

	// DPS/HPS
	val := p.Dps
	label := "DPS"
	if !isDD {
		val = p.Hps
		label = "HPS"
	}
	dc.SetRGB(1, 1, 1)
	dc.DrawString(fmt.Sprintf("%d %s", val, label), 500, y+25)

	// Spec Rank
	dc.SetRGB(1, 0.8, 0)
	dc.DrawString(fmt.Sprintf("#%d", p.SpecRank), 700, y+25)

	return y + rowHeight
}
