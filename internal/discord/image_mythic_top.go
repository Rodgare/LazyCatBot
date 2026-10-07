package discord

import (
	"LazyCatBot/internal/models"
	"LazyCatBot/internal/sirus"
	"bytes"
	"fmt"
	"math/rand"
	"strings"

	"github.com/fogleman/gg"
)

const (
	mtWidth       = 800
	mtRowHeight   = 40
	mtTableHeader = 30

	mtColRankX      = 20
	mtColClassX     = 45
	mtColSpecX      = 75
	mtColNameX      = 105
	mtColIlvlX      = 340
	mtColZodiacX    = 390
	mtColDiamondX   = 440
	mtColRioX       = 480
	mtColClassRankX = 550
	mtColSpecRankX  = 600
	mtColKeyX       = 650
	mtColRunsX      = 710
)

// RenderMythicGuildTopImage renders a mythic rating leaderboard table for guild
// members sorted by their current score (Rio).
func RenderMythicGuildTopImage(guildName, realm string, players []models.GuildMythicPlayer) ([]byte, error) {
	if len(players) == 0 {
		return nil, nil
	}

	headerH := 110
	rowsTotalHeight := float64(len(players)) * mtRowHeight
	height := float64(headerH) + 10 + mtTableHeader + rowsTotalHeight + 20

	dc := gg.NewContext(mtWidth, int(height))

	drawMythicGuildBackground(dc)

	// Title
	loadMythicFontFace(dc, 22)
	title := fmt.Sprintf("🏆 Мифик-рейтинг: %s", guildName)
	dc.SetRGBA(0, 0, 0, 0.8)
	dc.DrawStringAnchored(title, mtWidth/2+1, 31, 0.5, 0.5)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(title, mtWidth/2, 30, 0.5, 0.5)

	// Subtitle
	loadMythicFontFace(dc, 14)
	dc.SetHexColor("#9aa0a8")
	sub := fmt.Sprintf("Игроков с рейтингом: %d   |   Сервер: %s", len(players), strings.ToUpper(realm))
	dc.DrawStringAnchored(sub, mtWidth/2, 62, 0.5, 0.5)

	y := float64(headerH) + 10
	drawMythicGuildTableHeader(dc, y)
	y += mtTableHeader

	for i := range players {
		y = drawMythicGuildPlayerRow(dc, i+1, players[i], len(players), y)
	}

	buf := new(bytes.Buffer)
	err := dc.EncodePNG(buf)
	return buf.Bytes(), err
}

var raidBgIDs = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46}

func drawMythicGuildBackground(dc *gg.Context) {
	raidID := raidBgIDs[rand.Intn(len(raidBgIDs))]
	drawRaidBackground(dc, raidID, "#0a0e14")
}

func drawMythicGuildTableHeader(dc *gg.Context, y float64) {
	dc.SetRGBA(0, 0, 0, 0.35)
	dc.DrawRectangle(0, y, float64(mtWidth), mtTableHeader)
	dc.Fill()

	loadMythicFontFace(dc, 16)
	dc.SetRGB(0.7, 0.7, 0.7)
	dc.DrawString("#", mtColRankX, y+20)
	dc.DrawString("Игрок", mtColNameX, y+20)
	dc.DrawString("ILvl", mtColIlvlX, y+20)
	drawMythicImage(dc, "assets/images/4b.png", float64(mtColDiamondX)-1, y-5)
	dc.DrawString("Рио", mtColRioX, y+20)
	dc.DrawString("Класс", mtColClassRankX-15, y+20)
	dc.DrawString("Спек", mtColSpecRankX-15, y+20)
	dc.DrawString("Ключ", mtColKeyX-15, y+20)
	dc.DrawString("Забегов", mtColRunsX-15, y+20)
}

func drawMythicGuildPlayerRow(dc *gg.Context, rank int, p models.GuildMythicPlayer, total int, y float64) float64 {
	if rank%2 == 0 {
		dc.SetRGBA(1, 1, 1, 0.03)
		dc.DrawRectangle(0, y, float64(mtWidth), mtRowHeight)
		dc.Fill()
	}

	loadMythicFontFace(dc, 16)

	// Rank
	dc.SetRGB(0.5, 0.5, 0.5)
	dc.DrawString(fmt.Sprintf("%d.", rank), mtColRankX, y+28)

	// Class icon
	drawMythicImage(dc, fmt.Sprintf("assets/images/class/%d.png", p.ClassID), mtColClassX, y)

	// Spec icon
	specName := sirus.GetSpecName(p.ClassID, p.SpecID)
	if specs, ok := specIcons[p.ClassID]; ok {
		if filename, ok2 := specs[specName]; ok2 {
			drawMythicImage(dc, "assets/images/spec/"+filename, mtColSpecX, y)
		}
	}

	// Name (class color)
	colorHex, ok := classColors[p.ClassID]
	if !ok {
		colorHex = "#FFFFFF"
	}
	dc.SetHexColor(colorHex)
	dc.DrawString(p.Name, mtColNameX, y+28)

	nameW, _ := dc.MeasureString(p.Name)
	if p.Title != "" {
		loadMythicFontFace(dc, 10)
		dc.SetHexColor("#8b93a1")
		dc.DrawString(" "+p.Title, mtColNameX+nameW, y+23)
	}

	// ILvl
	loadMythicFontFace(dc, 16)
	dc.SetHexColor("#ffffff")
	dc.DrawString(fmt.Sprintf("%d", p.Ilvl), mtColIlvlX, y+28)

	// Zodiac
	drawMythicImage(dc, fmt.Sprintf("assets/images/zodiac/%d.png", p.Zodiac), mtColZodiacX, y)

	// Black diamonds
	dc.SetHexColor("#ebb914")
	dc.DrawString(fmt.Sprintf("%d", p.BlackDiamonds), mtColDiamondX, y+28)

	// Rio colored by guild percentile
	percentile := 0
	if total > 0 {
		percentile = (total - rank + 1) * 100 / total
		if percentile > 100 {
			percentile = 100
		}
	}
	if percentile > 0 {
		r, g, b := getColorByPercentile(percentile)
		dc.SetRGB(r, g, b)
	} else {
		dc.SetHexColor("#d8a13c")
	}
	dc.DrawString(fmt.Sprintf("%.0f", p.Score), mtColRioX, y+28)

	// Class rank (colored by eliteness within the class)
	drawMythicGuildRank(dc, p.ClassRank, p.ClassTotal, mtColClassRankX, y)

	// Spec rank (colored by eliteness within the spec)
	drawMythicGuildRank(dc, p.SpecRank, p.SpecTotal, mtColSpecRankX, y)

	// Best key
	dc.SetHexColor("#d4dbe9")
	dc.DrawString(fmt.Sprintf("%d", p.BestKey), mtColKeyX, y+28)

	// Runs
	dc.DrawString(fmt.Sprintf("%d", p.TotalRuns), mtColRunsX, y+28)

	return y + mtRowHeight
}

// drawMythicGuildRank draws a leaderboard position colored by its percentile:
// the lower the position (smaller number), the more elite the color.
func drawMythicGuildRank(dc *gg.Context, rank, total int, x, y float64) {
	loadMythicFontFace(dc, 16)
	if rank <= 0 || total <= 0 {
		dc.SetHexColor("#8b93a1")
		dc.DrawString("-", x, y+28)
		return
	}
	percentile := (total - rank + 1) * 100 / total
	if percentile > 100 {
		percentile = 100
	}
	r, g, b := getColorByPercentile(percentile)
	dc.SetRGB(r, g, b)
	dc.DrawString(fmt.Sprintf("%d", rank), x, y+28)
}
