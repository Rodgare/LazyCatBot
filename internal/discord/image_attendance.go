package discord

import (
	"bytes"
	"fmt"
	"image"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"LazyCatBot/internal/models"
	"LazyCatBot/internal/sirus"

	"github.com/fogleman/gg"
)

const (
	attendanceWidth      = 1250
	attendancePad        = 30
	attendanceHeaderH    = 165
	attendanceTableHeadH = 42
	attendanceRowH       = 44
	attendanceFooterH    = 42

	attRankX     = 30
	attClassX    = 74
	attNameX     = 104
	attSpecIconX = 370
	attSpecTextX = 396
	attRoleX     = 545
	attIlvlX     = 625
	attSetX      = 695
	attKillsX    = 825
	attBarX      = 925
	attBarW      = 100
)

// RenderAttendanceImage draws the full guild attendance table for the period.
// Unlike the old embed it renders every player, class/spec icons, attendance
// bars, last-seen timestamps, black diamonds and titles.
func RenderAttendanceImage(guildName, realm string, since int64, total int, entries []models.AttendanceEntry) ([]byte, error) {
	height := attendanceHeaderH + attendanceTableHeadH + len(entries)*attendanceRowH + attendanceFooterH + attendancePad
	if height < 320 {
		height = 320
	}

	dc := gg.NewContext(attendanceWidth, height)
	if raidID := randomRaidBackgroundID(); raidID > 0 {
		drawRaidBackground(dc, raidID, getRaidColor(raidID))
	} else {
		dc.SetRGB(0.055, 0.065, 0.085)
		dc.Clear()
	}
	dc.SetRGBA(0.02, 0.03, 0.05, 0.28)
	dc.DrawRectangle(0, 0, attendanceWidth, float64(height))
	dc.Fill()

	// Gold accent line, same visual language as the report and digest images.
	dc.SetRGB(0.95, 0.77, 0.24)
	dc.DrawRectangle(0, 0, attendanceWidth, 4)
	dc.Fill()

	if guildName == "" {
		guildName = "Гильдия"
	}

	loadCyrillicFontFace(dc, 34)
	dc.SetRGB(0.96, 0.96, 0.96)
	dc.DrawString(fitText(dc, "Посещаемость — "+guildName, attendanceWidth-attendancePad*2), attendancePad, 58)

	avg := 0
	full := 0
	if total > 0 && len(entries) > 0 {
		sum := 0
		for _, e := range entries {
			if e.Kills > total {
				e.Kills = total
			}
			sum += e.Kills
			if e.Kills == total {
				full++
			}
		}
		avg = sum * 100 / (len(entries) * total)
	}

	loadCyrillicFontFace(dc, 18)
	dc.SetRGB(0.62, 0.66, 0.72)
	until := time.Now().Unix()
	dc.DrawString(fmt.Sprintf("%s · %s – %s · 4 КД · киллов: %d · игроков: %d · средняя явка: %d%% · 100%%: %d",
		upperRealm(realm), formatDigestDate(since), formatDigestDate(until), total, len(entries), avg, full),
		attendancePad, 92)

	dc.SetRGBA(0.95, 0.77, 0.24, 0.25)
	dc.DrawRectangle(attendancePad, 118, attendanceWidth-attendancePad*2, 1)
	dc.Fill()

	headerY := float64(attendanceHeaderH)
	dc.SetRGBA(0, 0, 0, 0.38)
	dc.DrawRectangle(0, headerY, attendanceWidth, attendanceTableHeadH)
	dc.Fill()
	loadCyrillicFontFace(dc, 15)
	dc.SetRGB(0.48, 0.52, 0.58)
	dc.DrawString("#", attRankX, headerY+27)
	dc.DrawString("Игрок", attNameX, headerY+27)
	dc.DrawString("Спек", attSpecTextX, headerY+27)
	dc.DrawString("Роль", attRoleX, headerY+27)
	dc.DrawString("Илвл", attIlvlX, headerY+27)
	dc.DrawString("Сет", attSetX, headerY+27)
	dc.DrawString("Килы", attKillsX, headerY+27)
	dc.DrawString("Явка", attBarX, headerY+27)
	dc.DrawString("Последний кил", 1105, headerY+27)
	dc.DrawString("ЧБ", 1210, headerY+27)

	dc.SetRGBA(1, 1, 1, 0.08)
	dc.DrawRectangle(attendancePad, headerY+attendanceTableHeadH-1, attendanceWidth-attendancePad*2, 1)
	dc.Fill()

	y := headerY + attendanceTableHeadH
	for idx, e := range entries {
		drawAttendanceRow(dc, idx, e, total, y)
		y += attendanceRowH
	}

	loadCyrillicFontFace(dc, 14)
	dc.SetRGB(0.36, 0.39, 0.45)
	dc.DrawString("LazyCatBot · посещаемость гильдии", attendancePad, float64(height)-18)

	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func drawAttendanceRow(dc *gg.Context, idx int, e models.AttendanceEntry, total int, y float64) {
	if idx%2 == 1 {
		dc.SetRGBA(0, 0, 0, 0.18)
		dc.DrawRectangle(0, y, attendanceWidth, attendanceRowH)
		dc.Fill()
	}

	rank := idx + 1
	loadCyrillicFontFace(dc, 16)
	switch rank {
	case 1:
		dc.SetRGB(0.95, 0.77, 0.24)
	case 2:
		dc.SetRGB(0.78, 0.80, 0.84)
	case 3:
		dc.SetRGB(0.80, 0.55, 0.32)
	default:
		dc.SetRGB(0.48, 0.52, 0.58)
	}
	dc.DrawString(fmt.Sprintf("%d", rank), attRankX, y+28)

	drawAttendanceAsset(dc, fmt.Sprintf("assets/images/class/%d.png", e.ClassID), attClassX, y+attendanceRowH/2-10)

	loadCyrillicFontFace(dc, 18)
	dc.SetColor(classColorRGBA(e.ClassID))
	nameY := y + 19
	if e.Title == "" {
		nameY = y + 28
	}
	dc.DrawString(fitText(dc, e.Name, 250), attNameX, nameY)
	if e.Title != "" {
		loadCyrillicFontFace(dc, 12)
		dc.SetRGB(0.55, 0.58, 0.64)
		dc.DrawString(fitText(dc, "«"+e.Title+"»", 250), attNameX, y+34)
	}

	specName := sirus.GetSpecName(e.ClassID, e.SpecID)
	specTextX := float64(attSpecTextX)
	if specs, ok := specIcons[e.ClassID]; ok {
		if file, ok := specs[specName]; ok {
			if w, ok := drawAttendanceAsset(dc, "assets/images/spec/"+file, attSpecIconX, y+attendanceRowH/2-10); ok {
				specTextX += w + 8
			}
		}
	}
	loadCyrillicFontFace(dc, 16)
	dc.SetRGB(0.82, 0.85, 0.90)
	dc.DrawString(fitText(dc, specName, 112), specTextX, y+19)
	loadCyrillicFontFace(dc, 11)
	dc.SetRGB(0.50, 0.54, 0.60)
	dc.DrawString(fitText(dc, className(e.ClassID), 112), specTextX, y+35)

	role, roleColor := attendanceRole(e.Role)
	loadCyrillicFontFace(dc, 16)
	dc.SetRGB(roleColor[0], roleColor[1], roleColor[2])
	dc.DrawString(role, attRoleX, y+28)

	loadCyrillicFontFace(dc, 16)
	dc.SetRGB(0.82, 0.85, 0.90)
	dc.DrawString(fmt.Sprintf("%d", e.Ilvl), attIlvlX, y+28)

	setText := e.SetPieces
	if setText == "" {
		setText = "—"
		dc.SetRGB(0.38, 0.41, 0.47)
	} else {
		dc.SetRGB(0.70, 0.25, 0.95)
	}
	loadCyrillicFontFace(dc, 14)
	dc.DrawString(fitText(dc, setText, 110), attSetX, y+28)

	kills := e.Kills
	if total > 0 && kills > total {
		kills = total
	}
	dc.SetRGB(0.92, 0.93, 0.95)
	dc.DrawString(fmt.Sprintf("%d / %d", kills, total), attKillsX, y+28)

	percent := 0
	if total > 0 {
		percent = kills * 100 / total
	}
	barY := y + attendanceRowH/2 - 5
	dc.SetRGBA(1, 1, 1, 0.08)
	dc.DrawRoundedRectangle(attBarX, barY, attBarW, 10, 5)
	dc.Fill()
	if percent > 0 {
		fillW := attBarW * float64(percent) / 100
		if fillW < 10 {
			fillW = 10
		}
		barColor := attendanceBarColor(percent)
		dc.SetRGB(barColor[0], barColor[1], barColor[2])
		dc.DrawRoundedRectangle(attBarX, barY, fillW, 10, 5)
		dc.Fill()
	}
	loadCyrillicFontFace(dc, 15)
	dc.SetRGB(0.76, 0.79, 0.84)
	dc.DrawString(fmt.Sprintf("%d%%", percent), attBarX+attBarW+12, y+28)

	loadCyrillicFontFace(dc, 15)
	dc.SetRGB(0.62, 0.66, 0.72)
	dc.DrawString(formatAttendanceTime(e.LastSeen), 1105, y+28)

	loadCyrillicFontFace(dc, 16)
	if e.BlackDiamonds > 0 {
		dc.SetRGB(0.92, 0.73, 0.25)
		dc.DrawString(fmt.Sprintf("%d", e.BlackDiamonds), 1210, y+28)
	} else {
		dc.SetRGB(0.38, 0.41, 0.47)
		dc.DrawString("—", 1210, y+28)
	}
}

func attendanceRole(role string) (string, [3]float64) {
	switch role {
	case "hps":
		return "Хил", [3]float64{0.42, 0.86, 0.55}
	case "tank":
		return "Танк", [3]float64{0.55, 0.72, 1.0}
	default:
		return "ДД", [3]float64{0.80, 0.83, 0.88}
	}
}

func attendanceBarColor(percent int) [3]float64 {
	switch {
	case percent >= 90:
		return [3]float64{0.35, 0.82, 0.45}
	case percent >= 75:
		return [3]float64{0.35, 0.65, 0.95}
	case percent >= 50:
		return [3]float64{0.62, 0.48, 0.92}
	case percent >= 25:
		return [3]float64{0.95, 0.62, 0.25}
	default:
		return [3]float64{0.85, 0.35, 0.35}
	}
}

func formatAttendanceTime(unix int64) string {
	if unix == 0 {
		return "—"
	}
	return time.Unix(unix, 0).In(time.FixedZone("MSK", 3*3600)).Format("02.01 15:04")
}

func drawAttendanceAsset(dc *gg.Context, path string, x, y float64) (float64, bool) {
	data, err := imagesFS.ReadFile(path)
	if err != nil {
		return 0, false
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return 0, false
	}
	dc.DrawImage(img, int(x), int(y))
	return float64(img.Bounds().Dx()), true
}

func fitText(dc *gg.Context, s string, maxWidth float64) string {
	if s == "" {
		return s
	}
	w, _ := dc.MeasureString(s)
	if w <= maxWidth {
		return s
	}
	runes := []rune(s)
	for len(runes) > 1 {
		runes = runes[:len(runes)-1]
		candidate := string(runes) + "…"
		w, _ = dc.MeasureString(candidate)
		if w <= maxWidth {
			return candidate
		}
	}
	return "…"
}

// randomRaidBackgroundID picks one of the embedded raid artworks so wide
// attendance/profile images share the visual style of kill reports.
func randomRaidBackgroundID() int {
	entries, err := imagesFS.ReadDir("assets/images/raid")
	if err != nil {
		return 0
	}
	ids := make([]int, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".png") {
			continue
		}
		id, err := strconv.Atoi(strings.TrimSuffix(entry.Name(), ".png"))
		if err == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return 0
	}
	return ids[rand.Intn(len(ids))]
}
