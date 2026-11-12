package discord

import (
	"bytes"
	"fmt"

	"LazyCatBot/internal/sirus"

	"github.com/fogleman/gg"
)

const (
	profileWidth   = 900
	profilePad     = 30
	profileHeaderH = 160
	profileRowH    = 44
	profileFooterH = 48

	profileLeftLabelX  = 30
	profileLeftValueX  = 235
	profileRightLabelX = 465
	profileRightValueX = 640
)

// ProfileCard is the pre-formatted data needed to draw a player profile image.
type ProfileCard struct {
	Name          string
	GuildName     string
	Realm         string
	ClassID       int
	SpecID        int
	Role          string
	Ilvl          int
	MythicValue   string
	BestKey       string
	Runs          string
	Position      string
	BlackDiamonds int
	Title         string
	Sets          string
	Attendance    string
	BestDps       int
	BestHps       int
	FirstSeen     int64
	LastSeen      int64
	TotalKills    int
}

type profileStat struct {
	label string
	value string
}

// RenderPlayerProfileImage draws a card with the full profile: character data,
// mythic progress, raid attendance and kill timestamps.
func RenderPlayerProfileImage(c ProfileCard) ([]byte, error) {
	left := []profileStat{
		{"Класс и спек", className(c.ClassID) + " · " + sirus.GetSpecName(c.ClassID, c.SpecID)},
		{"Роль", orDash(c.Role)},
		{"Илвл", formatPositive(c.Ilvl)},
		{"Т4 / сеты", orDash(c.Sets)},
		{"Чёрные бриллианты", formatPositive(c.BlackDiamonds)},
		{"Титул", orDash(c.Title)},
		{"Посещаемость", orDash(c.Attendance)},
		{"Первый килл", formatProfileTime(c.FirstSeen)},
	}
	right := []profileStat{
		{"Rio (мифик)", orDash(c.MythicValue)},
		{"Лучший ключ", orDash(c.BestKey)},
		{"Мифик-раны", orDash(c.Runs)},
		{"Позиция в ладдере", orDash(c.Position)},
		{"Лучший DPS", formatPositive(c.BestDps)},
		{"Лучший HPS", formatPositive(c.BestHps)},
		{"Последний килл", formatProfileTime(c.LastSeen)},
		{"Килов за 4 КД", formatPositive(c.TotalKills)},
	}

	height := profileHeaderH + len(left)*profileRowH + profileFooterH + profilePad
	dc := gg.NewContext(profileWidth, height)
	dc.SetRGB(0.055, 0.065, 0.085)
	dc.Clear()

	classColor := classColorRGBA(c.ClassID)
	dc.SetRGB(float64(classColor.R)/255, float64(classColor.G)/255, float64(classColor.B)/255)
	dc.DrawRectangle(0, 0, profileWidth, 6)
	dc.Fill()

	loadCyrillicFontFace(dc, 14)
	dc.SetRGB(0.45, 0.49, 0.56)
	dc.DrawString("ПРОФИЛЬ ИГРОКА", profilePad, 46)

	loadCyrillicFontFace(dc, 40)
	dc.SetRGB(0.96, 0.96, 0.96)
	dc.SetColor(classColor)
	dc.DrawString(fitText(dc, c.Name, 600), profilePad, 94)

	loadCyrillicFontFace(dc, 17)
	dc.SetRGB(0.62, 0.66, 0.72)
	subtitle := fmt.Sprintf("%s · %s · 4 КД", upperRealm(c.Realm), c.GuildName)
	if c.GuildName == "" {
		subtitle = fmt.Sprintf("%s · 4 КД", upperRealm(c.Realm))
	}
	dc.DrawString(fitText(dc, subtitle, 650), profilePad, 124)

	// Ilvl badge on the right side of the header.
	dc.SetRGBA(float64(classColor.R)/255, float64(classColor.G)/255, float64(classColor.B)/255, 0.16)
	dc.DrawRoundedRectangle(profileWidth-205, 42, 175, 78, 12)
	dc.Fill()
	loadCyrillicFontFace(dc, 13)
	dc.SetRGB(0.62, 0.66, 0.72)
	dc.DrawString("ILVL", profileWidth-185, 66)
	loadCyrillicFontFace(dc, 32)
	dc.SetRGB(0.96, 0.96, 0.96)
	dc.DrawString(formatPositive(c.Ilvl), profileWidth-185, 102)

	y := float64(profileHeaderH)
	for i := 0; i < len(left); i++ {
		if i%2 == 1 {
			dc.SetRGBA(1, 1, 1, 0.03)
			dc.DrawRectangle(profilePad, y, profileWidth-profilePad*2, profileRowH)
			dc.Fill()
		}
		drawProfileStat(dc, left[i], profileLeftLabelX, profileLeftValueX, y)
		drawProfileStat(dc, right[i], profileRightLabelX, profileRightValueX, y)
		y += profileRowH
	}

	dc.SetRGBA(1, 1, 1, 0.06)
	dc.DrawRectangle(profileRightLabelX-14, profileHeaderH, 1, float64(len(left)*profileRowH))
	dc.Fill()

	loadCyrillicFontFace(dc, 14)
	dc.SetRGB(0.36, 0.39, 0.45)
	dc.DrawString("LazyCatBot · профиль игрока · последние 4 КД", profilePad, float64(height)-18)

	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func drawProfileStat(dc *gg.Context, stat profileStat, labelX, valueX, y float64) {
	loadCyrillicFontFace(dc, 14)
	dc.SetRGB(0.52, 0.56, 0.63)
	dc.DrawString(stat.label, labelX, y+27)

	loadCyrillicFontFace(dc, 17)
	dc.SetRGB(0.90, 0.92, 0.95)
	maxWidth := float64(profileRightLabelX - profileLeftValueX - 20)
	if labelX >= profileRightLabelX {
		maxWidth = float64(profileWidth - profileRightValueX - profilePad)
	}
	dc.DrawString(fitText(dc, stat.value, maxWidth), valueX, y+27)
}

func formatPositive(v int) string {
	if v <= 0 {
		return "—"
	}
	return fmt.Sprintf("%d", v)
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
