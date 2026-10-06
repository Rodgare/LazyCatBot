package discord

import (
	"os"
	"sync"

	"github.com/fogleman/gg"
)

var embeddedFontFiles = []string{
	"assets/fonts/DroidSans-Bold.ttf",
	"assets/fonts/Roboto-Regular.ttf",
}

var fallbackFontPaths = []string{
	"internal/discord/assets/fonts/Roboto-Regular.ttf",
	"C:\\Windows\\Fonts\\arialbd.ttf",
	"C:\\Windows\\Fonts\\arial.ttf",
	"C:\\Windows\\Fonts\\seguiemj.ttf",
	"/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
	"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
	"/usr/share/fonts/truetype/freefont/FreeSansBold.ttf",
	"/usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf",
	"/usr/share/fonts/dejavu/DejaVuSans.ttf",
}

var (
	embeddedFontPathOnce sync.Once
	embeddedFontPathVal  string
)

func embeddedFontPath() string {
	embeddedFontPathOnce.Do(func() {
		for _, p := range embeddedFontFiles {
			data, err := imagesFS.ReadFile(p)
			if err != nil {
				continue
			}
			f, err := os.CreateTemp("", "lazycabot-font-*.ttf")
			if err != nil {
				continue
			}
			if _, err := f.Write(data); err != nil {
				f.Close()
				os.Remove(f.Name())
				continue
			}
			f.Close()
			embeddedFontPathVal = f.Name()
			return
		}
	})
	return embeddedFontPathVal
}

func loadCyrillicFontFace(dc *gg.Context, size float64) bool {
	if path := embeddedFontPath(); path != "" {
		if err := dc.LoadFontFace(path, size); err == nil {
			return true
		}
	}

	for _, path := range fallbackFontPaths {
		if _, err := os.Stat(path); err == nil {
			if err := dc.LoadFontFace(path, size); err == nil {
				return true
			}
		}
	}
	return false
}