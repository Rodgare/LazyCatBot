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

const (
	width           = 800
	rowHeight       = 35
	headerH         = 50
	margin          = 20
	colRankX        = 20
	colCategoryX    = 45
	colNameX        = 70
	colIlvlX        = 230
	colZodiacX      = 270
	ColSetX         = 310
	colDpsX         = 350
	colIlvlRankX    = 430
	colSpecRankX    = 530
	colClassRankX   = 630
	colOverallRankX = 730
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

//go:embed assets/images
var imagesFS embed.FS

func RenderReportImage(report BossKillReport) ([]byte, error) {
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

	// Title
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(fmt.Sprintf("%s — %s", report.MapName, report.BossName), width/2, y+20, 0.5, 0.5)
	y += headerH

	if len(dds) > 0 {
		dc.DrawString("Дпс", colDpsX, y+20)
		y = drawRoleHeader(dc, "Дамагеры", y, width)
		for i, p := range dds {
			y = drawPlayerRow(dc, i+1, p, y, width, rowHeight, true)
		}

	}

	if len(healers) > 0 {
		y += 10
		dc.SetRGB(1, 1, 1)
		dc.DrawString("Хпс", colDpsX, y+20)
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
	dc.DrawString(title, colRankX, y+20)
	dc.DrawString("ILvl", colIlvlX, y+20)
	dc.DrawString("Созв", colZodiacX-7, y+20)
	dc.DrawString("t4", ColSetX, y+20)
	dc.DrawString("Спек/Илвл", colIlvlRankX-10, y+20)
	dc.DrawString("Спек", colSpecRankX, y+20)
	dc.DrawString("Класс", colClassRankX, y+20)
	dc.DrawString("Все", colOverallRankX, y+20)
	return y + 35
}

func drawPlayerRow(dc *gg.Context, rank int, p PlayerReport, y float64, width int, rowHeight float64, isDD bool) float64 {
	//rank
	if rank%2 == 0 {
		dc.SetRGBA(1, 1, 1, 0.03)
		dc.DrawRectangle(0, y, float64(width), rowHeight)
		dc.Fill()
	}

	dc.SetRGB(0.5, 0.5, 0.5)
	dc.DrawString(fmt.Sprintf("%d.", rank), colRankX, y+25)

	//name
	colorHex, ok := classColors[p.ClassID]
	if !ok {
		colorHex = "#FFFFFF"
	}
	dc.SetHexColor(colorHex)
	dc.DrawString(p.Name, colNameX, y+25)

	//zodiac
	zodiacFilePath := fmt.Sprintf("assets/images/zodiac/%d.png", p.Zodiac)
	zodiacFileData, err := imagesFS.ReadFile(zodiacFilePath)
	if err == nil {
		img, _, errDecode := image.Decode(bytes.NewReader(zodiacFileData))
		if errDecode == nil {
			bounds := img.Bounds()
			imgH := bounds.Dy()
			imgY := int(y) + int(rowHeight)/2 - imgH/2
			dc.DrawImage(img, colZodiacX, imgY)
		} else {
			fmt.Printf("Error decoding %s: %v\n", zodiacFilePath, errDecode)
		}
	} else {
		fmt.Printf("Error: file %s not found in embed\n", zodiacFilePath)
	}

	//category
	categoryImgName := getCategoryImageName(p.Category)
	categoryFilePath := "assets/images/category/" + categoryImgName
	categoryFileData, err := imagesFS.ReadFile(categoryFilePath)
	if err == nil {
		img, _, errDecode := image.Decode(bytes.NewReader(categoryFileData))
		if errDecode == nil {
			bounds := img.Bounds()
			imgH := bounds.Dy()
			imgY := int(y) + int(rowHeight)/2 - imgH/2
			dc.DrawImage(img, colCategoryX, imgY+3)
		} else {
			fmt.Printf("Error decoding %s: %v\n", categoryFilePath, errDecode)
		}
	} else {
		fmt.Printf("Error: file %s not found in embed\n", categoryFilePath)
	}

	//ilvl rank
	ilvlRankTextX := float64(colIlvlRankX)

	if specs, ok := specIcons[p.ClassID]; ok {
		if filename, ok2 := specs[p.SpecName]; ok2 {
			filePath := "assets/images/spec/" + filename

			fileData, err := imagesFS.ReadFile(filePath)
			if err == nil {
				img, _, errDecode := image.Decode(bytes.NewReader(fileData))
				if errDecode == nil {
					bounds := img.Bounds()
					imgW := bounds.Dx()
					imgH := bounds.Dy()

					imgY := int(y) + int(rowHeight)/2 - imgH/2
					dc.DrawImage(img, colIlvlRankX, imgY)

					ilvlRankTextX += float64(imgW + 10)
				} else {
					fmt.Printf("Error decoding %s: %v\n", filePath, errDecode)
				}
			} else {
				fmt.Printf("Error: file %s not found in embed\n", filePath)
			}
		}
	}

	r, g, b := getColorByPercentile(p.IlvlPercentile)
	dc.SetRGB(r, g, b)
	dc.DrawString(fmt.Sprintf("%d", p.IlvlRank), ilvlRankTextX, y+25)

	//SpecRank
	specTextX := float64(colSpecRankX)

	if specs, ok := specIcons[p.ClassID]; ok {
		if filename, ok2 := specs[p.SpecName]; ok2 {
			filePath := "assets/images/spec/" + filename

			fileData, err := imagesFS.ReadFile(filePath)
			if err == nil {
				img, _, errDecode := image.Decode(bytes.NewReader(fileData))
				if errDecode == nil {
					bounds := img.Bounds()
					imgW := bounds.Dx()
					imgH := bounds.Dy()

					imgY := int(y) + int(rowHeight)/2 - imgH/2
					dc.DrawImage(img, colSpecRankX, imgY)

					specTextX += float64(imgW + 10)
				} else {
					fmt.Printf("Error decoding %s: %v\n", filePath, errDecode)
				}
			} else {
				fmt.Printf("Error: file %s not found in embed\n", filePath)
			}
		}
	}

	r, g, b = getColorByPercentile(p.SpecPercentile)
	dc.SetRGB(r, g, b)
	dc.DrawString(fmt.Sprintf("%d", p.SpecRank), specTextX, y+25)

	//ClassRank
	classTextX := float64(colClassRankX)

	filename := fmt.Sprintf("%d.png", p.ClassID)
	filePath := "assets/images/class/" + filename

	fileData, err := imagesFS.ReadFile(filePath)
	if err == nil {
		img, _, errDecode := image.Decode(bytes.NewReader(fileData))
		if errDecode == nil {
			bounds := img.Bounds()
			imgW := bounds.Dx()
			imgH := bounds.Dy()

			imgY := int(y) + int(rowHeight)/2 - imgH/2
			dc.DrawImage(img, colClassRankX, imgY)

			classTextX += float64(imgW + 10)
		} else {
			fmt.Printf("Error decoding %s: %v\n", filePath, errDecode)
		}
	} else {
		fmt.Printf("Error: file %s not found in embed\n", filePath)
	}
	r, g, b = getColorByPercentile(p.ClassPercentile)
	dc.SetRGB(r, g, b)
	dc.DrawString(fmt.Sprintf("%d", p.ClassRank), classTextX, y+25)

	//overall rank
	r, g, b = getColorByPercentile(p.OverallPercentile)
	dc.SetRGB(r, g, b)
	dc.DrawString(fmt.Sprintf("%d", p.OverallRank), colOverallRankX, y+25)

	//ilvl
	dc.SetRGB(0.8, 0.8, 0.8)
	dc.DrawString(fmt.Sprintf("%d", p.Ilvl), colIlvlX, y+25)

	//T4
	dc.SetRGB(0.7, 0.2, 1.0)
	dc.DrawString(fmt.Sprintf("%d", p.T4), ColSetX+5, y+25)

	val := p.Dps
	if !isDD {
		val = p.Hps
	}
	dc.SetRGB(1, 1, 1)
	dc.DrawString(fmt.Sprintf("%s", FormatNum(val)), colDpsX, y+25)

	// dc.SetRGB(1, 0.8, 0.2)
	// dc.DrawString(fmt.Sprintf("#%d", p.SpecRank), colClassRankX, y+25)

	return y + rowHeight
}

func getColorByPercentile(p int) (float64, float64, float64) {
	switch {
	case p >= 100:
		return 0.90, 0.80, 0.50 //золото/песочный (#e5cc80)
	case p >= 99:
		return 0.89, 0.41, 0.66 // розовый (#e268a8)
	case p >= 95:
		return 1.00, 0.50, 0.00 // оранжевый (#ff8000)
	case p >= 75:
		return 0.64, 0.21, 0.93 // фиолетовый (#a335ee)
	case p >= 50:
		return 0.00, 0.44, 1.00 // синий (#0070ff)
	case p >= 25:
		return 0.12, 1.00, 0.00 // зелёный (#1eff00)
	default:
		return 0.40, 0.40, 0.40 // серый (#666666)
	}
}

func getCategoryImageName(cat int) string {
	switch {
	case cat <= 90002:
		return "custom_category_7.png"
	case cat <= 90005:
		return "custom_category_6.png"
	case cat <= 90009:
		return "custom_category_5.png"
	case cat <= 90014:
		return "custom_category_4.png"
	case cat <= 90020:
		return "custom_category_3.png"
	case cat <= 90027:
		return "custom_category_2.png"
	case cat <= 90035:
		return "custom_category_1.png"
	default:
		return "custom_category_ooc_1.png"
	}
}
