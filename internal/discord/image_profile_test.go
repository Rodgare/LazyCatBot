package discord

import (
	"bytes"
	"testing"
)

func TestRenderPlayerProfileImage(t *testing.T) {
	card := ProfileCard{
		Name:          "Алиса",
		GuildName:     "Тестовая гильдия",
		Realm:         "x3",
		ClassID:       8,
		SpecID:        1,
		Role:          "ДД",
		Ilvl:          260,
		MythicValue:   "3200 · топ 5% (12/250)",
		BestKey:       "+10",
		Runs:          "18/24",
		Position:      "#12",
		BlackDiamonds: 3,
		Title:         "Хранитель пламени",
		Sets:          "4т4 4т6",
		Attendance:    "90% (9 килов)",
		BestDps:       123456,
		BestHps:       0,
		FirstSeen:     1762300000,
		LastSeen:      1762900000,
		TotalKills:    10,
	}

	img, err := RenderPlayerProfileImage(card)
	if err != nil {
		t.Fatalf("render profile: %v", err)
	}
	if len(img) < 1000 {
		t.Fatalf("profile image too small: %d bytes", len(img))
	}
	if !bytes.HasPrefix(img, []byte{0x89, 'P', 'N', 'G'}) {
		t.Fatalf("profile is not a PNG")
	}
	writePreview(t, "profile_preview.png", img)
}
