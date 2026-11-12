package discord

import (
	"image/color"
	"math/rand"
	"strings"
	"sync"

	"github.com/fogleman/gg"
)

// Donor nicknames are matched case-insensitively and realm-agnostically: if a
// name is marked premium it is highlighted in every report on every realm.
var (
	premiumMu    sync.RWMutex
	premiumNames = make(map[string]struct{})
)

func normalizeDonorName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// SetPremiumNames replaces the in-memory donor set (used on startup from DB).
func SetPremiumNames(names []string) {
	m := make(map[string]struct{}, len(names))
	for _, n := range names {
		k := normalizeDonorName(n)
		if k != "" {
			m[k] = struct{}{}
		}
	}
	premiumMu.Lock()
	premiumNames = m
	premiumMu.Unlock()
}

// AddPremiumName registers a single donor nickname (used by the webhook).
func AddPremiumName(name string) {
	k := normalizeDonorName(name)
	if k == "" {
		return
	}
	premiumMu.Lock()
	premiumNames[k] = struct{}{}
	premiumMu.Unlock()
}

// IsPremiumName reports whether the nickname belongs to a donor.
func IsPremiumName(name string) bool {
	k := normalizeDonorName(name)
	if k == "" {
		return false
	}
	premiumMu.RLock()
	_, ok := premiumNames[k]
	premiumMu.RUnlock()
	return ok
}

// shouldHighlight reports whether a row is drawn with the premium backdrop:
// real donors always are, plus one "showcase" player per report so everyone can
// see what the premium nickname looks like.
func shouldHighlight(name, showcase string) bool {
	if IsPremiumName(name) {
		return true
	}
	if showcase == "" {
		return false
	}
	return normalizeDonorName(name) == normalizeDonorName(showcase)
}

// pickShowcaseName chooses a random nickname to highlight as a teaser. If the
// report already contains a real donor, it returns "" so only actual donors are
// highlighted and no random player is touched.
func pickShowcaseName(names []string) string {
	candidates := make([]string, 0, len(names))
	for _, n := range names {
		if strings.TrimSpace(n) == "" {
			continue
		}
		if IsPremiumName(n) {
			return ""
		}
		candidates = append(candidates, n)
	}
	if len(candidates) == 0 {
		return ""
	}
	return candidates[rand.Intn(len(candidates))]
}

const (
	premiumRowInset  = 2.0
	premiumRowRadius = 5.0
	premiumRowAccent = 3.0
)

// premiumTint keeps the row tint close to the class color, but darkens very
// light colors (priest white, rogue yellow) so the translucent backdrop never
// washes out the nickname drawn on top of it.
func premiumTint(c color.RGBA) color.RGBA {
	lum := 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
	if lum <= 200 {
		return c
	}
	const f = 0.55
	return color.RGBA{
		R: uint8(float64(c.R) * f),
		G: uint8(float64(c.G) * f),
		B: uint8(float64(c.B) * f),
		A: 255,
	}
}

// drawPremiumRow paints the donor highlight across the whole table row using
// the player's class color. The gradient is strongest around the rank and
// nickname, fades through the middle and reaches full transparency at the
// right edge, so numbers and icons stay readable. A thin left accent bar and a
// hairline inner border separate the row from its neighbours; the small inset
// keeps stacked donor rows from merging.
func drawPremiumRow(dc *gg.Context, y, h, width float64, classColor color.RGBA) {
	x := premiumRowInset
	w := width - premiumRowInset*2
	yy := y + premiumRowInset
	hh := h - premiumRowInset*2
	if w <= 0 || hh <= 0 {
		return
	}

	tint := premiumTint(classColor)
	stop := func(a uint8) color.RGBA {
		return color.RGBA{R: tint.R, G: tint.G, B: tint.B, A: a}
	}

	grad := gg.NewLinearGradient(x, 0, x+w, 0)
	grad.AddColorStop(0, stop(64))
	grad.AddColorStop(0.25, stop(46))
	grad.AddColorStop(0.55, stop(22))
	grad.AddColorStop(1, stop(0))
	dc.SetFillStyle(grad)
	dc.DrawRoundedRectangle(x, yy, w, hh, premiumRowRadius)
	dc.Fill()

	br := float64(classColor.R) + (255-float64(classColor.R))*0.35
	bg := float64(classColor.G) + (255-float64(classColor.G))*0.35
	bb := float64(classColor.B) + (255-float64(classColor.B))*0.35
	dc.SetLineWidth(1)
	dc.SetRGBA(br/255, bg/255, bb/255, 0.20)
	dc.DrawRoundedRectangle(x+0.5, yy+0.5, w-1, hh-1, premiumRowRadius)
	dc.Stroke()

	dc.SetRGBA(float64(tint.R)/255, float64(tint.G)/255, float64(tint.B)/255, 0.90)
	dc.DrawRoundedRectangle(x+1, yy+5, premiumRowAccent, hh-10, 1.5)
	dc.Fill()
}
