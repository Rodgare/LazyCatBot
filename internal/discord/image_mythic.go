package discord

import (
	"LazyCatBot/internal/models"
	"LazyCatBot/internal/sirus"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/fogleman/gg"
	xdraw "golang.org/x/image/draw"
)

const (
	mWidth       = 800
	mScale       = 2
	mRowHeight   = 40
	mTableHeader = 30
	mMargin      = 20

	mColRankX   = 20
	mColClassX  = 45
	mColSpecX   = 75
	mColNameX   = 105
	mColRoleX   = 280
	mColIlvlX   = 365
	mColZodiacX = 440
	mColDpsX    = 510
	mColHpsX    = 620
	mColIntX    = 720
)

var mythicColors = map[int]string{
	4:  "#0a0e14",
	5:  "#0e0c16",
	6:  "#10131a",
	8:  "#0b0f1a",
	9:  "#140f0a",
	10: "#0a1410",
	11: "#120a14",
	13: "#0a0a0e",
}

var mythicFontPaths = []string{
	"internal/discord/assets/fonts/Roboto-Regular.ttf",
	"C:\\Windows\\Fonts\\arialbd.ttf",
	"C:\\Windows\\Fonts\\arial.ttf",
	"/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
	"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
	"/usr/share/fonts/truetype/freefont/FreeSansBold.ttf",
	"/usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf",
	"/usr/share/fonts/dejavu/DejaVuSans.ttf",
}

func getMythicColor(challengeID int) string {
	if col, ok := mythicColors[challengeID]; ok {
		return col
	}
	return "#0c1014"
}

func loadMythicFontFace(dc *gg.Context, size float64) {
	for _, path := range mythicFontPaths {
		if _, err := os.Stat(path); err == nil {
			if err := dc.LoadFontFace(path, size); err == nil {
				return
			}
		}
	}
	slog.Warn("No suitable font found for Cyrillic support")
}

func (r *DiscordReporter) RenderMythicReportImage(report *models.MythicReport) ([]byte, error) {
	if len(report.Members) == 0 {
		return nil, nil
	}

	if report.HasCombat {
		combatByGUID := make(map[int]int, len(report.CombatStats))
		for _, c := range report.CombatStats {
			combatByGUID[c.GUID] = c.DamageDone
		}
		slices.SortFunc(report.Members, func(a, b models.ReportMember) int {
			return combatByGUID[b.MemberGUID] - combatByGUID[a.MemberGUID]
		})
	}

	headerH := 100
	if report.HasCombat {
		headerH = 125
	}

	rowsTotalHeight := float64(len(report.Members)) * mRowHeight
	height := float64(headerH) + 10 + mTableHeader + rowsTotalHeight + 20

	dc := gg.NewContext(mWidth*mScale, int(height)*mScale)
	dc.Scale(mScale, mScale)

	drawMythicBackground(dc, report)

	// Title (Dungeon Name + Level)
	loadMythicFontFace(dc, 14*mScale)
	title := fmt.Sprintf("%s — %d", report.Name, report.KeystoneLevel)
	titleW, _ := dc.MeasureString(title)
	titleW /= mScale

	dc.SetRGBA(0, 0, 0, 0.8)
	// dc.DrawStringAnchored(title, mWidth/2+1, 31, 0.5, 0.5)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(title, mWidth/2, 25, 0.5, 0.5)

	// Bonus levels (closed key in time)
	loadMythicFontFace(dc, 12*mScale)
	bonus := report.RewardLevel
	bonusText := fmt.Sprintf("+%d", report.RewardLevel)
	bonusX := mWidth/2 + titleW
	dc.SetRGBA(0, 0, 0, 0.8)
	dc.SetHexColor(getBonusColor(bonus))
	dc.DrawStringAnchored(bonusText, bonusX, 28, 0, 0.2)

	// Status badge (in time / not in time)
	statusStr := "В тайм"
	statusColor := "#2ecc71"
	if report.RewardLevel <= 0 {
		statusStr = "Не в тайм"
		statusColor = "#e74c3c"
	}
	loadMythicFontFace(dc, 10*mScale)
	dc.SetHexColor(statusColor)
	dc.DrawStringAnchored(statusStr, mWidth-mMargin+10, 20, 1.0, 0.5)

	// Details Subtitle (Timer, Score, Deaths)
	loadMythicFontFace(dc, 9*mScale)
	dc.SetHexColor("#9aa0a8")
	min := report.Timer / 60
	sec := report.Timer % 60
	subText := fmt.Sprintf("Время: %02d:%02d   |   Очки: %.2f   |   Смертей: %d", min, sec, report.Score, report.DeathCounter)
	dc.DrawStringAnchored(subText, mWidth/2, 56, 0.5, 0.5)

	// Affixes
	var affixNames []string
	for _, affixID := range report.Affixes {
		if affix, ok := models.AffixMap[affixID]; ok {
			affixNames = append(affixNames, affix.Name)
		}
	}
	if len(affixNames) > 0 {
		loadMythicFontFace(dc, 9*mScale)
		dc.SetHexColor("#d8a13c")
		dc.DrawStringAnchored(fmt.Sprintf("Аффиксы: %s", strings.Join(affixNames, " • ")), mWidth/2, 85, 0.5, 0.5)
	}

	// Total DPS / HPS / Kicks summary
	if report.HasCombat {
		var totalDPS, totalHPS, totalKicks int
		for _, c := range report.CombatStats {
			totalDPS += c.DamageDone
			totalHPS += c.HealDone
			totalKicks += c.Interrupts
		}

		loadMythicFontFace(dc, 9*mScale)
		dc.SetHexColor("#8f7bc9")
		combatText := fmt.Sprintf("Суммарный ДПС: %s   |   ХПС: %s   |   Кики: %d",
			FormatNum(totalDPS), FormatNum(totalHPS), totalKicks)
		dc.DrawStringAnchored(combatText, mWidth/2, 115, 0.5, 0.5)
	}

	// Table header
	y := float64(headerH) + 10
	drawMythicTableHeader(dc, y, report.HasCombat)
	y += mTableHeader

	combatByGUID := make(map[int]models.MythicMemberCombat, len(report.CombatStats))
	for _, c := range report.CombatStats {
		combatByGUID[c.GUID] = c
	}

	for i, member := range report.Members {
		y = drawMythicPlayerRow(dc, i+1, member, combatByGUID[member.MemberGUID], y, report.HasCombat)
	}

	buf := new(bytes.Buffer)
	full := dc.Image()
	dst := image.NewRGBA(image.Rect(0, 0, mWidth, int(height)))
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), full, full.Bounds(), xdraw.Over, nil)
	err := png.Encode(buf, dst)
	return buf.Bytes(), err
}

func getBonusColor(bonus int) string {
	switch {
	case bonus >= 2:
		return "#2ecc71" // green — key closed well in time
	case bonus == 1:
		return "#f1c40f" // gold — key closed on the edge
	default:
		return "#e74c3c" // red — key not closed in time
	}
}

func drawMythicBackground(dc *gg.Context, report *models.MythicReport) {
	bgColor := getMythicColor(report.ChallengeID)
	dc.SetHexColor(bgColor + "ff")
	dc.Clear()

	filePath := fmt.Sprintf("assets/images/mythic/%d.png", report.ChallengeID)
	fileData, err := imagesFS.ReadFile(filePath)
	if err != nil {
		return
	}

	img, _, errDecode := image.Decode(bytes.NewReader(fileData))
	if errDecode != nil {
		fmt.Printf("Error decoding mythic image %s: %v\n", filePath, errDecode)
		return
	}

	dc.DrawImage(img, 0, 0)

	grad := gg.NewLinearGradient(0, 0, 0, 400)

	var r, g, b int
	fmt.Sscanf(bgColor, "#%02x%02x%02x", &r, &g, &b)
	bgR, bgG, bgB := uint8(r), uint8(g), uint8(b)

	grad.AddColorStop(0, color.RGBA{0, 0, 0, 160})
	grad.AddColorStop(0.5, color.RGBA{0, 0, 0, 110})
	grad.AddColorStop(1.0, color.RGBA{bgR, bgG, bgB, 190})

	dc.SetFillStyle(grad)
	dc.DrawRectangle(0, 0, float64(mWidth), 400)
	dc.Fill()

	dc.SetHexColor(bgColor + "ff")
	dc.DrawRectangle(0, 400, float64(mWidth), 5000)
	dc.Fill()
}

func drawMythicTableHeader(dc *gg.Context, y float64, combat bool) {
	dc.SetRGBA(0, 0, 0, 0.35)
	dc.DrawRectangle(0, y, float64(mWidth), mTableHeader)
	dc.Fill()

	loadMythicFontFace(dc, 10*mScale)
	dc.SetRGB(0.7, 0.7, 0.7)
	dc.DrawString("#", mColRankX, y+20)
	dc.DrawString("Игрок", mColNameX, y+20)
	dc.DrawString("Роль", mColRoleX, y+20)
	dc.DrawString("ILvl", mColIlvlX, y+20)
	if combat {
		dc.DrawString("Созв", mColZodiacX-15, y+20)
		dc.DrawString("Дпс", mColDpsX, y+20)
		dc.DrawString("Хпс", mColHpsX, y+20)
		dc.DrawString("Кики", mColIntX, y+20)
	}
}

func drawMythicPlayerRow(dc *gg.Context, rank int, member models.ReportMember, combat models.MythicMemberCombat, y float64, hasCombat bool) float64 {
	if rank%2 == 0 {
		dc.SetRGBA(1, 1, 1, 0.03)
		dc.DrawRectangle(0, y, float64(mWidth), mRowHeight)
		dc.Fill()
	}

	loadMythicFontFace(dc, 10*mScale)

	// Rank
	dc.SetRGB(0.5, 0.5, 0.5)
	dc.DrawString(fmt.Sprintf("%d.", rank), mColRankX, y+28)

	// Class icon
	drawMythicImage(dc, fmt.Sprintf("assets/images/class/%d.png", member.ClassID), mColClassX, y)

	// Spec icon
	specName := sirus.GetSpecName(member.ClassID, member.SpecID)
	if specs, ok := specIcons[member.ClassID]; ok {
		if filename, ok2 := specs[specName]; ok2 {
			drawMythicImage(dc, "assets/images/spec/"+filename, mColSpecX, y)
		}
	}

	// Name (class color)
	colorHex, ok := classColors[member.ClassID]
	if !ok {
		colorHex = "#FFFFFF"
	}
	dc.SetHexColor(colorHex)
	dc.DrawString(member.Name, mColNameX, y+28)

	// Role
	loadMythicFontFace(dc, 9*mScale)
	roleText := "Неизвестно"
	switch member.RoleID {
	case 1:
		roleText = "Танк"
	case 2:
		roleText = "Хил"
	case 3:
		roleText = "ДПС"
	}
	dc.SetHexColor("#8b93a1")
	dc.DrawString(roleText, mColRoleX, y+28)

	// Item level
	dc.SetHexColor("#c8ccd4")
	dc.DrawString(fmt.Sprintf("%d", member.Ilvl), mColIlvlX, y+28)

	// Zodiac icon
	drawMythicImage(dc, fmt.Sprintf("assets/images/zodiac/%d.png", member.Zodiac), mColZodiacX, y)

	// DPS / HPS / Interrupts from the run log
	if hasCombat {
		dc.SetRGB(1, 1, 1)
		if combat.GUID != 0 {
			dc.DrawString(FormatNum(combat.DamageDone), mColDpsX, y+28)
			dc.DrawString(FormatNum(combat.HealDone), mColHpsX, y+28)
			dc.DrawString(fmt.Sprintf("%d", combat.Interrupts), mColIntX, y+28)
		} else {
			dc.DrawString("-", mColDpsX, y+28)
			dc.DrawString("-", mColHpsX, y+28)
			dc.DrawString("-", mColIntX, y+28)
		}
	}

	return y + mRowHeight
}

func drawMythicImage(dc *gg.Context, path string, x, y float64) {
	fileData, err := imagesFS.ReadFile(path)
	if err != nil {
		return
	}

	img, _, errDecode := image.Decode(bytes.NewReader(fileData))
	if errDecode != nil {
		fmt.Printf("Error decoding %s: %v\n", path, errDecode)
		return
	}

	bounds := img.Bounds()
	imgH := bounds.Dy()
	imgY := int(y) + int(mRowHeight)/2 - imgH/2
	dc.DrawImage(img, int(x), imgY)
}
