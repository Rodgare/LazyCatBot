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

// premiumTint keeps the row tint a darker, slightly richer version of the
// class color so the bright nickname drawn on top never blends into its own
// backdrop. Very light colors are pulled down further.
func premiumTint(c color.RGBA) color.RGBA {
	const base = 0.70
	r := float64(c.R) * base
	g := float64(c.G) * base
	b := float64(c.B) * base

	lum := 0.299*r + 0.587*g + 0.114*b
	if lum > 150 {
		f := 150 / lum
		r, g, b = r*f, g*f, b*f
	}
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
}

// drawPremiumRow paints the donor highlight across the whole table row using
// the player's class color. The gradient is a soft smoky glow: it peaks around
// the rank and nickname at ~18% opacity, fades to ~4% by the middle and is
// fully transparent before the right-side columns, so numbers and icons keep
// their own colors. A 3px accent bar in the pure class color marks the row and
// a hairline inner border separates it from its neighbours; the small inset
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
	grad.AddColorStop(0, stop(46))
	grad.AddColorStop(0.22, stop(30))
	grad.AddColorStop(0.50, stop(12))
	grad.AddColorStop(0.80, stop(0))
	dc.SetFillStyle(grad)
	dc.DrawRoundedRectangle(x, yy, w, hh, premiumRowRadius)
	dc.Fill()

	br := float64(classColor.R) + (255-float64(classColor.R))*0.30
	bg := float64(classColor.G) + (255-float64(classColor.G))*0.30
	bb := float64(classColor.B) + (255-float64(classColor.B))*0.30
	dc.SetRGBA(br/255, bg/255, bb/255, 0.16)
	dc.DrawRectangle(x+6, yy+0.5, w-12, 1)
	dc.Fill()
	dc.DrawRectangle(x+6, yy+hh-1.5, w-12, 1)
	dc.Fill()

	dc.SetRGBA(float64(classColor.R)/255, float64(classColor.G)/255, float64(classColor.B)/255, 1)
	dc.DrawRoundedRectangle(x+1, yy+4, premiumRowAccent, hh-8, 1.5)
	dc.Fill()
}

// drawNameShadow draws a soft dark drop shadow under a nickname so the
// class-colored text stays readable on top of the class-tinted row.
func drawNameShadow(dc *gg.Context, s string, x, y float64) {
	dc.SetRGBA(0, 0, 0, 0.90)
	dc.DrawString(s, x, y+1.5)
	dc.SetRGBA(0, 0, 0, 0.35)
	dc.DrawString(s, x, y+2.5)
}
