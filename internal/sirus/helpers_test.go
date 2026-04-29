package sirus

import (
	"fmt"
	"testing"
	"time"
)

// --- IsPlayerKillToday ---
// Принимает формат: "2026-04-29T13:25:54.000000Z" (RFC3339 / ISO 8601 UTC)
// Сравнивает с текущим днём в зоне MSK (UTC+3)

func TestIsPlayerKillToday(t *testing.T) {
	loc := time.FixedZone("MSK", 3*3600)
	now := time.Now().In(loc)

	// Сегодня 12:00 MSK → UTC = MSK-3h
	todayNoon := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, loc)
	// Вчера 12:00 MSK
	yesterdayNoon := todayNoon.AddDate(0, 0, -1)
	// Завтра 12:00 MSK
	tomorrowNoon := todayNoon.AddDate(0, 0, 1)
	// Сегодня ровно в полночь MSK (00:00:00) — граничный случай: НЕ после начала дня
	todayMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	// Сегодня 00:00:01 MSK — уже после полуночи
	todayJustAfterMidnight := todayMidnight.Add(time.Second)

	formatRFC3339Micro := func(t time.Time) string {
		// Конвертируем в UTC и форматируем как "2006-01-02T15:04:05.000000Z"
		utc := t.UTC()
		return fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d.000000Z",
			utc.Year(), utc.Month(), utc.Day(),
			utc.Hour(), utc.Minute(), utc.Second())
	}

	tests := []struct {
		name     string
		dateStr  string
		want     bool
	}{
		{
			name:    "сегодняшний кил в полдень MSK — true",
			dateStr: formatRFC3339Micro(todayNoon),
			want:    true,
		},
		{
			name:    "вчерашний кил — false",
			dateStr: formatRFC3339Micro(yesterdayNoon),
			want:    false,
		},
		{
			name:    "завтрашний кил — true (после начала сегодняшнего дня)",
			dateStr: formatRFC3339Micro(tomorrowNoon),
			want:    true,
		},
		{
			name:    "ровно полночь MSK сегодня — false (не строго после)",
			dateStr: formatRFC3339Micro(todayMidnight),
			want:    false,
		},
		{
			name:    "секунда после полуночи MSK сегодня — true",
			dateStr: formatRFC3339Micro(todayJustAfterMidnight),
			want:    true,
		},
		{
			name:    "невалидная строка — false",
			dateStr: "not-a-date",
			want:    false,
		},
		{
			name:    "пустая строка — false",
			dateStr: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPlayerKillToday(tt.dateStr)
			if got != tt.want {
				t.Errorf("IsPlayerKillToday(%q) = %v, want %v", tt.dateStr, got, tt.want)
			}
		})
	}
}

// --- IsGuildKillToday ---
// Принимает формат: "2026-04-25 21:55:40" (считается временем MSK UTC+3)
// Сравнивает с текущим днём в зоне MSK

func TestIsGuildKillToday(t *testing.T) {
	loc := time.FixedZone("MSK", 3*3600)
	now := time.Now().In(loc)

	todayNoon := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, loc)
	yesterdayNoon := todayNoon.AddDate(0, 0, -1)
	tomorrowNoon := todayNoon.AddDate(0, 0, 1)
	todayMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	todayJustAfterMidnight := todayMidnight.Add(time.Second)

	layout := "2006-01-02 15:04:05"

	tests := []struct {
		name    string
		dateStr string
		want    bool
	}{
		{
			name:    "сегодняшний кил в полдень MSK — true",
			dateStr: todayNoon.Format(layout),
			want:    true,
		},
		{
			name:    "вчерашний кил — false",
			dateStr: yesterdayNoon.Format(layout),
			want:    false,
		},
		{
			name:    "завтрашний кил — true (после начала сегодняшнего дня)",
			dateStr: tomorrowNoon.Format(layout),
			want:    true,
		},
		{
			name:    "ровно полночь сегодня MSK — false (не строго после)",
			dateStr: todayMidnight.Format(layout),
			want:    false,
		},
		{
			name:    "секунда после полуночи сегодня MSK — true",
			dateStr: todayJustAfterMidnight.Format(layout),
			want:    true,
		},
		{
			name:    "невалидная строка — false",
			dateStr: "not-a-date",
			want:    false,
		},
		{
			name:    "пустая строка — false",
			dateStr: "",
			want:    false,
		},
		{
			name:    "неверный формат (RFC3339) — false",
			dateStr: "2026-04-29T13:25:54.000000Z",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsGuildKillToday(tt.dateStr)
			if got != tt.want {
				t.Errorf("IsGuildKillToday(%q) = %v, want %v", tt.dateStr, got, tt.want)
			}
		})
	}
}
