package discord

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"os"
	"slices"

	"github.com/fogleman/gg"
)

const (
	width           = 800
	rowHeight       = 35
	headerH         = 50
	footerH         = 20
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

var raidColors = map[int]string{
	1:  "#020000",
	2:  "#020000",
	3:  "#0c1011",
	4:  "#0c1011",
	5:  "#121018",
	6:  "#121018",
	7:  "#0c1011",
	8:  "#0c1011",
	9:  "#080a0e",
	10: "#080a0e",
	11: "#0c0d12",
	12: "#080c0e",
	13: "#080c0e",
	14: "#03060a",
	15: "#03060a",
	16: "#080801",
	17: "#010000",
	18: "#03060a",
	19: "#03060a",
	20: "#0c0d12",
	21: "#080801",
	22: "#010000",
	23: "#090a0d",
	24: "#090a0d",
	25: "#0a0c0e",
	26: "#0a0c0e",
	27: "#09090d",
	28: "#0c0607",
	29: "#0c0607",
	30: "#0a0c0e",
	31: "#0a0c0e",
	32: "#0c0607",
	33: "#0c0607",
	34: "#070404",
	35: "#070404",
	36: "#0a030d",
	37: "#0a1013",
	38: "#0a030d",
	39: "#0a1013",
	40: "#080a0e",
	41: "#080a0e",
	42: "#070a0c",
	43: "#070a0c",
	44: "#0f0c0f",
	45: "#0f0c0f",
	46: "#020700",
}

func getRaidColor(raidID int) string {
	if col, ok := raidColors[raidID]; ok {
		return col
	}
	return "#151618" // Дефолтный фон
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
	height := headerH + footerH + (totalRows+2)*rowHeight + margin*2

	dc := gg.NewContext(width, int(height))

	// Фон
	bgColor := getRaidColor(report.RaidOrder)
	dc.SetHexColor(bgColor + "ff")
	dc.Clear()

	// Картинка рейда (800px)
	drawRaidBackground(dc, report.RaidOrder, bgColor)

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
	// Добавим небольшую тень для заголовка, чтобы он читался на фоне картинки
	dc.SetRGBA(0, 0, 0, 0.8)
	dc.DrawStringAnchored(fmt.Sprintf("%s — %s", report.MapName, report.BossName), width/2+1, y+21, 0.5, 0.5)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(fmt.Sprintf("%s — %s", report.MapName, report.BossName), width/2, y+20, 0.5, 0.5)

	y += headerH
	//dd
	if len(dds) > 0 {
		dc.DrawString("Дпс", colDpsX, y+20)
		y = drawRoleHeader(dc, "Дамагеры", y, width)
		for i, p := range dds {
			y = drawPlayerRow(dc, i+1, p, y, width, rowHeight, true)
		}

	}
	//heals
	if len(healers) > 0 {
		y += 10
		dc.SetRGB(1, 1, 1)
		dc.DrawString("Хпс", colDpsX, y+20)
		y = drawRoleHeader(dc, "Хилы", y, width)
		for i, p := range healers {
			y = drawPlayerRow(dc, i+1, p, y, width, rowHeight, false)
		}

	}

	//footer
	dc.SetRGB(0.4, 0.4, 0.4)
	dc.DrawString("*Рейтинг за текущее кд", 570, y+20)

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
			dc.DrawImage(img, colZodiacX+4, imgY)
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
			dc.DrawImage(img, colCategoryX+2, imgY+2)
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

func drawRaidBackground(dc *gg.Context, raidID int, bgColor string) {
	filePath := fmt.Sprintf("assets/images/raid/%d.png", raidID)
	fileData, err := imagesFS.ReadFile(filePath)
	if err != nil {
		return // Если картинки нет в embed, просто выходим
	}

	img, _, errDecode := image.Decode(bytes.NewReader(fileData))
	if errDecode != nil {
		fmt.Printf("Error decoding raid image %s: %v\n", filePath, errDecode)
		return
	}

	// 1. Рисуем картинку в шапке (она 400px высотой)
	dc.DrawImage(img, 0, 0)

	// 2. --- ЕДИНЫЙ БЕСШОВНЫЙ ГРАДИЕНТ (Затемнение + Переход) ---
	// Этот градиент до середины просто затемняет, а потом плавно уводит в фон
	grad := gg.NewLinearGradient(0, 0, 0, 400)

	// Парсим HEX в RGBA для градиента
	var r, g, b int
	fmt.Sscanf(bgColor, "#%02x%02x%02x", &r, &g, &b)
	bgR, bgG, bgB := uint8(r), uint8(g), uint8(b)

	// От 0 до 200px: Держим стабильное затемнение (оверлей)
	grad.AddColorStop(0, color.RGBA{0, 0, 0, 200})   // 80% черного вверху
	grad.AddColorStop(0.5, color.RGBA{0, 0, 0, 200}) // Держим ту же темноту до 200px

	// От 200px до 400px: Плавно переходим из темноты в цвет фона
	grad.AddColorStop(1.0, color.RGBA{bgR, bgG, bgB, 255})

	dc.SetFillStyle(grad)
	dc.DrawRectangle(0, 0, float64(width), 400)
	dc.Fill()

	// 3. --- ЗАГЛУШКА (Закрашиваем всё, что ниже 400px) ---
	dc.SetHexColor(bgColor + "ff")
	dc.DrawRectangle(0, 400, float64(width), 5000) // С большим запасом вниз
	dc.Fill()
}
