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
// readable layout: header, first kills, top DPS/HPS and attendance.
func RenderWeeklyDigest(d models.WeeklyDigest) ([]byte, error) {
	attendance := d.Attendance
	if len(attendance) > 10 {
		attendance = attendance[:10]
	}
	firstKills := d.FirstKills
	if len(firstKills) > 6 {
		firstKills = firstKills[:6]
	}

	height := 150 + 20
	height += digestSectH + maxInt(1, len(firstKills))*digestRowH
	height += digestSectH + 5*digestRowH
	height += digestSectH + maxInt(1, len(attendance))*digestRowH
	height += 50

	dc := gg.NewContext(digestWidth, height)
	dc.SetRGB(0.06, 0.07, 0.09)
	dc.Clear()

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
	dc.DrawString(fmt.Sprintf("%s · %s – %s · килов: %d",
		upperRealm(d.Realm), formatDigestDate(d.Since), formatDigestDate(d.Until), d.TotalKills),
		digestPad, y)
	y += 34

	y = digestSection(dc, "Новые боссы", y)
	if len(firstKills) == 0 {
		y = digestEmptyRow(dc, "Новых боссов за неделю нет", y)
	} else {
		for _, fk := range firstKills {
			name := fk.BossName
			if name == "" {
				name = fmt.Sprintf("Босс %d", fk.BossID)
			}
			digestRowText(dc, name, formatDigestDateTime(fk.KilledAt), y)
			y += digestRowH
		}
	}
	y += 14

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

	y = digestSection(dc, "Посещаемость", y)
	if len(attendance) == 0 {
		y = digestEmptyRow(dc, "Нет данных о посещаемости", y)
	} else {
		for i, e := range attendance {
			percent := 0
			if d.TotalKills > 0 {
				percent = e.Kills * 100 / d.TotalKills
				if percent > 100 {
					percent = 100
				}
			}
			loadCyrillicFontFace(dc, 18)
			dc.SetRGB(0.55, 0.58, 0.64)
			dc.DrawString(fmt.Sprintf("%d.", i+1), digestPad, y+20)
			dc.SetColor(classColorRGBA(e.ClassID))
			dc.DrawString(e.Name, digestPad+32, y+20)
			loadCyrillicFontFace(dc, 18)
			dc.SetRGB(0.80, 0.83, 0.88)
			dc.DrawString(fmt.Sprintf("%d / %d · %d%%", e.Kills, d.TotalKills, percent),
				600, y+20)
			if e.BlackDiamonds > 0 {
				dc.SetRGB(0.55, 0.58, 0.64)
				dc.DrawString(fmt.Sprintf("ЧБ %d", e.BlackDiamonds), 790, y+20)
			}
			y += digestRowH
		}
	}

	y += 26
	loadCyrillicFontFace(dc, 15)
	dc.SetRGB(0.38, 0.41, 0.47)
	dc.DrawString("LazyCatBot · премиум-дайджест", digestPad, y)

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

func digestRowText(dc *gg.Context, left, right string, y float64) {
	loadCyrillicFontFace(dc, 18)
	dc.SetRGB(0.88, 0.90, 0.93)
	dc.DrawString(left, digestPad, y+20)
	dc.SetRGB(0.55, 0.58, 0.64)
	dc.DrawString(right, float64(digestWidth-digestPad-170), y+20)
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

func formatDigestDate(unix int64) string {
	if unix == 0 {
		return "—"
	}
	return time.Unix(unix, 0).In(time.FixedZone("MSK", 3*3600)).Format("02.01")
}

func formatDigestDateTime(unix int64) string {
	if unix == 0 {
		return "—"
	}
	return time.Unix(unix, 0).In(time.FixedZone("MSK", 3*3600)).Format("02.01 15:04")
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
