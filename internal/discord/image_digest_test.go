package discord

import (
	"bytes"
	"testing"

	"LazyCatBot/internal/models"
)

func TestRenderWeeklyDigest(t *testing.T) {
	digest := models.WeeklyDigest{
		GuildName:  "Тестовая гильдия",
		Realm:      "x3",
		Since:      1762300000,
		Until:      1762900000,
		TotalKills: 10,
		FirstKills: []models.FirstKill{
			{BossID: 1, BossName: "Тестовый босс", KilledAt: 1762400000},
		},
		TopDps: []models.DigestPlayer{
			{Name: "Alice", ClassID: 8, Value: 12345},
		},
		TopHps: []models.DigestPlayer{
			{Name: "Bob", ClassID: 5, Value: 9000},
		},
		Attendance: []models.AttendanceEntry{
			{Name: "Alice", ClassID: 8, Kills: 9, Ilvl: 260},
			{Name: "Bob", ClassID: 5, Kills: 8, Ilvl: 255},
		},
	}

	img, err := RenderWeeklyDigest(digest)
	if err != nil {
		t.Fatalf("render digest: %v", err)
	}
	if len(img) < 1000 {
		t.Fatalf("digest image too small: %d bytes", len(img))
	}
	if !bytes.HasPrefix(img, []byte{0x89, 'P', 'N', 'G'}) {
		t.Fatalf("digest is not a PNG")
	}
}
