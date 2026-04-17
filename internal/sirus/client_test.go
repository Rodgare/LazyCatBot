package sirus

import (
	"testing"
	"time"
)

func TestCalculateSirusDates(t *testing.T) {
	fixedTime := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)

	from, to := calculateSirusDates(fixedTime)

	wantTo := "2026-04-16"
	wantFrom := "2026-03-19"

	if to != wantTo {
		t.Errorf("calculateSirusDates() to = %v, want %v", to, wantTo)
	}
	if from != wantFrom {
		t.Errorf("calculateSirusDates() from = %v, want %v", from, wantFrom)
	}
}

func TestCalculateSirusDates_Wednesday(t *testing.T) {
	fixedTime := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)

	from, to := calculateSirusDates(fixedTime)

	wantTo := "2026-04-15"
	wantFrom := "2026-03-12"

	if to != wantTo {
		t.Errorf("Wednesday: calculateSirusDates() to = %v, want %v", to, wantTo)
	}
	if from != wantFrom {
		t.Errorf("Wednesday: calculateSirusDates() from = %v, want %v", from, wantFrom)
	}
}
