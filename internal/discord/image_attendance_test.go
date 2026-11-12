package discord

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"LazyCatBot/internal/models"
)

func TestRenderAttendanceImage(t *testing.T) {
	entries := []models.AttendanceEntry{
		{Name: "Алиса", ClassID: 8, SpecID: 1, Role: "dps", Kills: 10, Ilvl: 260, SetPieces: "4т4 4т6", BlackDiamonds: 3, Title: "Хранитель пламени", LastSeen: 1762900000},
		{Name: "Боб", ClassID: 5, SpecID: 1, Role: "hps", Kills: 9, Ilvl: 255, SetPieces: "4т4", BlackDiamonds: 1, LastSeen: 1762890000},
		{Name: "Василиса", ClassID: 2, SpecID: 0, Role: "hps", Kills: 8, Ilvl: 258, Title: "Свет несущая", LastSeen: 1762880000},
		{Name: "Григорий", ClassID: 1, SpecID: 2, Role: "dps", Kills: 7, Ilvl: 252, LastSeen: 1762870000},
		{Name: "Дмитрий", ClassID: 6, SpecID: 0, Role: "dps", Kills: 6, Ilvl: 250, LastSeen: 1762860000},
		{Name: "Ева", ClassID: 11, SpecID: 2, Role: "hps", Kills: 5, Ilvl: 248, LastSeen: 1762850000},
		{Name: "Жора", ClassID: 4, SpecID: 0, Role: "dps", Kills: 4, Ilvl: 245, LastSeen: 1762840000},
		{Name: "Зоя", ClassID: 9, SpecID: 2, Role: "dps", Kills: 3, Ilvl: 242, LastSeen: 1762830000},
		{Name: "Игорь", ClassID: 7, SpecID: 1, Role: "dps", Kills: 2, Ilvl: 240, LastSeen: 1762820000},
		{Name: "Ксения", ClassID: 3, SpecID: 0, Role: "dps", Kills: 1, Ilvl: 238, LastSeen: 1762810000},
	}

	img, err := RenderAttendanceImage("Тестовая гильдия", "x3", 1762300000, 10, entries)
	if err != nil {
		t.Fatalf("render attendance: %v", err)
	}
	if len(img) < 1000 {
		t.Fatalf("attendance image too small: %d bytes", len(img))
	}
	if !bytes.HasPrefix(img, []byte{0x89, 'P', 'N', 'G'}) {
		t.Fatalf("attendance is not a PNG")
	}
	writePreview(t, "attendance_preview.png", img)
}

func writePreview(t *testing.T, name string, img []byte) {
	t.Helper()
	if os.Getenv("RENDER_PREVIEW") == "" {
		return
	}
	path := filepath.Join(os.TempDir(), name)
	if err := os.WriteFile(path, img, 0o644); err != nil {
		t.Fatalf("write preview: %v", err)
	}
	t.Logf("preview: %s", path)
}
