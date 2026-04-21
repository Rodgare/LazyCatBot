package discord

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	_ "image/png"
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
	12: "#A330C9", // DH
}

var specIcons = map[int]map[string]string{
	1:  {"Arms": "WarA.png", "Fury": "WarF.png", "Prot": "WarP.png"},
	2:  {"Holy": "PaladinH.png", "Prot": "PaladinP.png", "Retri": "PaladinR.png"},
	3:  {"BM": "HunterBM.png", "MM": "HunterMM.png", "Surv": "HunterS.png"},
	4:  {"Assa": "RogueA.png", "Combat": "RogueC.png", "Sub": "RogueS.png"},
	5:  {"Disc": "PriestD.png", "Holy": "PriestH.png", "Shadow": "PriestS.png"},
	6:  {"Blood": "DKB.png", "Frost": "DKF.png", "Unholy": "DKU.png"},
	7:  {"Ele": "ShamanEl.png", "Enh": "ShamanEnch.png", "Resto": "ShamanR.png"},
	8:  {"Arcane": "MageA.png", "Fire": "MageF.png", "Frost": "MageFr.png"},
	9:  {"Affli": "LockA.png", "Demo": "LockDemon.png", "Destro": "LockDestr.png"},
	11: {"Bal": "DruidB.png", "Feral": "DruidF.png", "Resto": "DruidR.png", "Grd": "DruidF.png"},
}

//go:embed assets/images/*.png
var imagesFS embed.FS

func RenderReportImage(report BossKillReport) ([]byte, error) {
	const (
		width     = 800
		rowHeight = 35
		headerH   = 50
		margin    = 20
	)

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

	totalRows := len(dds) + len(healers)
	height := headerH + (totalRows+2)*rowHeight + margin*2

	dc := gg.NewContext(width, int(height))

	// Фон
	dc.SetHexColor("#151618ff")
	dc.Clear()

	fontLoaded := false
	fontPaths := []string{
		"internal/discord/assets/fonts/Roboto-Regular.ttf",             // Локальный шрифт из репозитория!
		"C:\\Windows\\Fonts\\arialbd.ttf",                              // Windows Arial Bold
		"C:\\Windows\\Fonts\\arial.ttf",                                // Windows Arial
		"C:\\Windows\\Fonts\\seguiemj.ttf",                             // Windows Segoe UI
		"/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",         // Linux Ubuntu/Debian
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",              // Linux Ubuntu/Debian
		"/usr/share/fonts/truetype/freefont/FreeSansBold.ttf",          // Linux FreeFont
		"/usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf", // Linux CentOS/RHEL
		"/usr/share/fonts/dejavu/DejaVuSans.ttf",                       // Alpine Linux
	}

	for _, path := range fontPaths {
		if _, err := os.Stat(path); err == nil {
			if err := dc.LoadFontFace(path, 16); err == nil {
				fontLoaded = true
				break
			}
		}
	}

	if !fontLoaded {
		fmt.Println("WARNING: No suitable font found for Cyrillic support!")
	}

	y := float64(margin)

	// Заголовок
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(fmt.Sprintf("%s — %s", report.MapName, report.BossName), width/2, y+20, 0.5, 0.5)
	y += headerH

	if len(dds) > 0 {
		y = drawRoleHeader(dc, "Дамагеры", y, width)
		for i, p := range dds {
			y = drawPlayerRow(dc, i+1, p, y, width, rowHeight, true)
		}
	}

	if len(healers) > 0 {
		y += 10
		y = drawRoleHeader(dc, "Хилы", y, width)
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
	dc.DrawString("Спек", 220, y+20)
	dc.DrawString("iLvl", 350, y+20)
	dc.DrawString("Дпс/Хпс", 500, y+20)
	dc.DrawString("Топ спек", 650, y+20)
	return y + 35
}

func drawPlayerRow(dc *gg.Context, rank int, p PlayerReport, y float64, width int, rowHeight float64, isDD bool) float64 {
	if rank%2 == 0 {
		dc.SetRGBA(1, 1, 1, 0.03)
		dc.DrawRectangle(0, y, float64(width), rowHeight)
		dc.Fill()
	}

	dc.SetRGB(0.5, 0.5, 0.5)
	dc.DrawString(fmt.Sprintf("%d.", rank), 20, y+25)

	colorHex, ok := classColors[p.ClassID]
	if !ok {
		colorHex = "#FFFFFF"
	}
	dc.SetHexColor(colorHex)
	dc.DrawString(p.Name, 55, y+25)

	specTextX := float64(220)

	if specs, ok := specIcons[p.ClassID]; ok {
		if filename, ok2 := specs[p.SpecName]; ok2 {
			filePath := "assets/images/" + filename

			fileData, err := imagesFS.ReadFile(filePath)
			if err == nil {
				img, _, errDecode := image.Decode(bytes.NewReader(fileData))
				if errDecode == nil {
					bounds := img.Bounds()
					imgW := bounds.Dx()
					imgH := bounds.Dy()

					imgY := int(y) + int(rowHeight)/2 - imgH/2
					dc.DrawImage(img, 220, imgY)

					specTextX += float64(imgW + 10)
				} else {
					fmt.Printf("Error decoding %s: %v\n", filePath, errDecode)
				}
			} else {
				fmt.Printf("Error: file %s not found in embed\n", filePath)
			}
		}
	}

	dc.SetRGB(0.6, 0.6, 0.6)
	dc.DrawString(p.SpecName, specTextX, y+25)

	dc.SetRGB(0.8, 0.8, 0.8)
	dc.DrawString(fmt.Sprintf("%d", p.Ilvl), 350, y+25)

	val := p.Dps
	label := "DPS"
	if !isDD {
		val = p.Hps
		label = "HPS"
	}
	dc.SetRGB(1, 1, 1)
	dc.DrawString(fmt.Sprintf("%s %s", FormatNum(val), label), 500, y+25)

	dc.SetRGB(1, 0.8, 0.2)
	dc.DrawString(fmt.Sprintf("#%d", p.SpecRank), 650, y+25)

	return y + rowHeight
}
