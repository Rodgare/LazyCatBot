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

// drawPremiumBackdrop paints a gold→purple gradient rounded rectangle behind a
// donor's nickname to set it apart from the regular rows.
func drawPremiumBackdrop(dc *gg.Context, x, y, w, h float64) {
	if w <= 0 || h <= 0 {
		return
	}
	grad := gg.NewLinearGradient(x, y, x+w, y+h)
	grad.AddColorStop(0, color.RGBA{R: 255, G: 199, B: 77, A: 120})
	grad.AddColorStop(1, color.RGBA{R: 156, G: 92, B: 255, A: 120})
	dc.SetFillStyle(grad)
	dc.DrawRoundedRectangle(x, y, w, h, 6)
	dc.Fill()

	dc.SetLineWidth(1)
	dc.SetRGBA(1, 0.86, 0.45, 0.65)
	dc.DrawRoundedRectangle(x, y, w, h, 6)
	dc.Stroke()
}
