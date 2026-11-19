package discord

import (
	"LazyCatBot/internal/models"
	"bytes"
	"fmt"
	"time"

	"github.com/fogleman/gg"
)

const (
	digestWidth = 900
	digestPad   = 30
	digestRowH  = 30
	digestSectH = 42
)

// RenderWeeklyDigest draws a self-contained weekly summary image. It uses only
// the embedded font/assets already shipped with the bot and keeps a simple,
// readable layout: header, top DPS/HPS, mythic and attendance.
func RenderWeeklyDigest(d models.WeeklyDigest) ([]byte, error) {
	attendance := d.Attendance
	if len(attendance) > 30 {
		attendance = attendance[:30]
	}
	mythic := d.MythicTop
	if len(mythic) > 10 {
		mythic = mythic[:10]
	}
	attendanceTotal := d.AttendanceKills
	if attendanceTotal <= 0 {
		attendanceTotal = d.TotalKills
	}

	mythicRows := (len(mythic) + 1) / 2
	attendanceRows := (len(attendance) + 1) / 2

	height := 130
	height += digestSectH + 5*digestRowH
	if len(mythic) > 0 {
		height += digestSectH + mythicRows*digestRowH
	}
	height += digestSectH + maxInt(1, attendanceRows)*digestRowH
	height += 60

	dc := gg.NewContext(digestWidth, height)
	if raidID := randomRaidBackgroundID(); raidID > 0 {
		drawRaidBackground(dc, raidID, getRaidColor(raidID))
	} else {
		dc.SetRGB(0.06, 0.07, 0.09)
		dc.Clear()
	}
	dc.SetRGBA(0.02, 0.03, 0.05, 0.30)
	dc.DrawRectangle(0, 0, digestWidth, float64(height))
	dc.Fill()

	dc.SetRGB(0.95, 0.77, 0.24)
	dc.DrawRectangle(0, 0, digestWidth, 4)
	dc.Fill()

	y := float64(digestPad)
	loadCyrillicFontFace(dc, 34)
	dc.SetRGB(0.96, 0.96, 0.96)
	title := fmt.Sprintf("%s — итоги недели", d.GuildName)
	if d.GuildName == "" {
		title = "Итоги недели"
	}
	dc.DrawString(title, digestPad, y+4)
	y += 44

	loadCyrillicFontFace(dc, 18)
	dc.SetRGB(0.62, 0.66, 0.72)
	killsText := fmt.Sprintf("килов: %d", d.TotalKills)
	if d.AttendanceKills > 0 && d.AttendanceKills != d.TotalKills {
		killsText += fmt.Sprintf(" · 25-плей: %d", d.AttendanceKills)
	}
	dc.DrawString(fmt.Sprintf("%s · %s – %s · %s",
		upperRealm(d.Realm), formatDigestDate(d.Since), formatDigestDate(d.Until), killsText),
		digestPad, y)
	y += 34

	y = digestSection(dc, "Топ DPS / Топ HPS", y)
	colW := float64(digestWidth-digestPad*2) / 2
	leftY, rightY := y, y
	for i := 0; i < 5; i++ {
		if i < len(d.TopDps) {
			digestPlayerRow(dc, float64(digestPad), leftY, d.TopDps[i], i+1)
		}
		if i < len(d.TopHps) {
			digestPlayerRow(dc, float64(digestPad)+colW+20, rightY, d.TopHps[i], i+1)
		}
		leftY += digestRowH
		rightY += digestRowH
	}
	y += 5*digestRowH + 14

	if len(mythic) > 0 {
		y = digestSection(dc, "Мифик", y)
		rowsPerCol := (len(mythic) + 1) / 2
		for i, p := range mythic {
			col := i / rowsPerCol
			row := i % rowsPerCol
			x := float64(digestPad) + float64(col)*(colW+20)
			digestMythicRow(dc, x, y+float64(row)*digestRowH, p, i+1)
		}
		y += float64(rowsPerCol)*digestRowH + 14
	}

	y = digestSection(dc, "Посещаемость", y)
	if len(attendance) == 0 {
		y = digestEmptyRow(dc, "Нет данных о посещаемости", y)
	} else {
		rowsPerCol := (len(attendance) + 1) / 2
		for i, e := range attendance {
			col := i / rowsPerCol
			row := i % rowsPerCol
			x := float64(digestPad) + float64(col)*(colW+20)
			digestAttendanceRow(dc, x, y+float64(row)*digestRowH, e, i+1, attendanceTotal)
		}
		y += float64(rowsPerCol) * digestRowH
	}

	y += 26
	loadCyrillicFontFace(dc, 15)
	dc.SetRGB(0.38, 0.41, 0.47)
	dc.DrawString("LazyCatBot · Посещаемость учитывает только актуальные рейды 25 игроков", digestPad, y)

	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func digestSection(dc *gg.Context, title string, y float64) float64 {
	loadCyrillicFontFace(dc, 21)
	dc.SetRGB(0.95, 0.77, 0.24)
	dc.DrawString(title, digestPad, y+20)
	dc.SetRGBA(0.95, 0.77, 0.24, 0.25)
	dc.DrawRectangle(digestPad, y+28, digestWidth-digestPad*2, 1)
	dc.Fill()
	return y + digestSectH
}

func digestEmptyRow(dc *gg.Context, text string, y float64) float64 {
	loadCyrillicFontFace(dc, 18)
	dc.SetRGB(0.48, 0.51, 0.57)
	dc.DrawString(text, digestPad, y+20)
	return y + digestRowH
}

func digestPlayerRow(dc *gg.Context, x, y float64, p models.DigestPlayer, rank int) {
	loadCyrillicFontFace(dc, 18)
	dc.SetRGB(0.55, 0.58, 0.64)
	dc.DrawString(fmt.Sprintf("%d.", rank), x, y+20)
	dc.SetColor(classColorRGBA(p.ClassID))
	dc.DrawString(p.Name, x+28, y+20)
	dc.SetRGB(0.82, 0.85, 0.90)
	dc.DrawString(FormatNum(p.Value), x+250, y+20)
	if p.BlackDiamonds > 0 {
		dc.SetRGB(0.55, 0.58, 0.64)
		dc.DrawString(fmt.Sprintf("ЧБ %d", p.BlackDiamonds), x+330, y+20)
	}
}

func digestMythicRow(dc *gg.Context, x, y float64, p models.GuildMythicPlayer, rank int) {
	loadCyrillicFontFace(dc, 18)
	dc.SetRGB(0.55, 0.58, 0.64)
	dc.DrawString(fmt.Sprintf("%d.", rank), x, y+20)
	dc.SetColor(classColorRGBA(p.ClassID))
	dc.DrawString(fitText(dc, p.Name, 210), x+28, y+20)
	dc.SetRGB(0.82, 0.85, 0.90)
	key := "—"
	if p.BestKey > 0 {
		key = fmt.Sprintf("+%d", p.BestKey)
	}
	dc.DrawString(fmt.Sprintf("%s  ·  %s", key, FormatNum(int(p.Score))), x+250, y+20)
}

func digestAttendanceRow(dc *gg.Context, x, y float64, e models.AttendanceEntry, rank, total int) {
	percent := 0
	if total > 0 {
		percent = e.Kills * 100 / total
		if percent > 100 {
			percent = 100
		}
	}
	loadCyrillicFontFace(dc, 18)
	dc.SetRGB(0.55, 0.58, 0.64)
	dc.DrawString(fmt.Sprintf("%d.", rank), x, y+20)
	dc.SetColor(classColorRGBA(e.ClassID))
	dc.DrawString(fitText(dc, e.Name, 210), x+32, y+20)
	dc.SetRGB(0.80, 0.83, 0.88)
	dc.DrawString(fmt.Sprintf("%d / %d · %d%%", e.Kills, total, percent), x+250, y+20)
}

func formatDigestDate(unix int64) string {
	if unix == 0 {
		return "—"
	}
	return time.Unix(unix, 0).In(time.FixedZone("MSK", 3*3600)).Format("02.01")
}

func upperRealm(realm string) string {
	if realm == "" {
		realm = "x3"
	}
	return "Sirus " + realm
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
