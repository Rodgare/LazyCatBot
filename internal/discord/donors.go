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

// drawPremiumRow paints the donor highlight across the whole table row. The
// gold gradient is strongest around the rank and nickname, fades through the
// middle and reaches full transparency at the right edge, so class colors and
// numbers stay readable. A thin left accent bar and a hairline inner border
// separate the row from its neighbours; the small inset keeps stacked donor
// rows from merging.
func drawPremiumRow(dc *gg.Context, y, h, width float64) {
	x := premiumRowInset
	w := width - premiumRowInset*2
	yy := y + premiumRowInset
	hh := h - premiumRowInset*2
	if w <= 0 || hh <= 0 {
		return
	}

	grad := gg.NewLinearGradient(x, 0, x+w, 0)
	grad.AddColorStop(0, color.RGBA{R: 245, G: 158, B: 11, A: 59})
	grad.AddColorStop(0.25, color.RGBA{R: 245, G: 158, B: 11, A: 42})
	grad.AddColorStop(0.55, color.RGBA{R: 245, G: 158, B: 11, A: 20})
	grad.AddColorStop(1, color.RGBA{R: 245, G: 158, B: 11, A: 0})
	dc.SetFillStyle(grad)
	dc.DrawRoundedRectangle(x, yy, w, hh, premiumRowRadius)
	dc.Fill()

	dc.SetLineWidth(1)
	dc.SetRGBA(1, 0.86, 0.50, 0.18)
	dc.DrawRoundedRectangle(x+0.5, yy+0.5, w-1, hh-1, premiumRowRadius)
	dc.Stroke()

	dc.SetRGBA(0.98, 0.72, 0.15, 0.90)
	dc.DrawRoundedRectangle(x+1, yy+5, premiumRowAccent, hh-10, 1.5)
	dc.Fill()
}
