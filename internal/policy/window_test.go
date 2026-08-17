package policy

import (
	"testing"
	"time"
)

func TestParseWindowBasic(t *testing.T) {
	w, err := ParseWindow("03:00-05:00", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if w.Start != 180 || w.End != 300 {
		t.Errorf("got start=%d end=%d, want 180 300", w.Start, w.End)
	}
}

func TestParseWindowAllDay(t *testing.T) {
	w, err := ParseWindow("00:00-00:00", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if !w.AllDay {
		t.Error("expected AllDay=true")
	}
}

func TestParseWindowOvernight(t *testing.T) {
	w, err := ParseWindow("23:00-01:00", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if !w.Overnight {
		t.Error("expected Overnight=true")
	}
}

func TestParseWindowEmpty(t *testing.T) {
	w, err := ParseWindow("", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if w != nil {
		t.Errorf("expected nil for empty window, got %+v", w)
	}
}

func TestWindowContains(t *testing.T) {
	w, _ := ParseWindow("03:00-05:00", "UTC")

	inside := time.Date(2026, 8, 17, 4, 30, 0, 0, time.UTC)
	if !w.Contains(inside) {
		t.Error("04:30 should be inside 03:00-05:00")
	}

	outside := time.Date(2026, 8, 17, 6, 0, 0, 0, time.UTC)
	if w.Contains(outside) {
		t.Error("06:00 should be outside 03:00-05:00")
	}
}

func TestWindowContainsBoundaries(t *testing.T) {
	w, _ := ParseWindow("03:00-05:00", "UTC")

	start := time.Date(2026, 8, 17, 3, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 17, 5, 0, 0, 0, time.UTC)

	if !w.Contains(start) {
		t.Error("03:00:00 should be inside (boundary included)")
	}
	if !w.Contains(end) {
		t.Error("05:00:00 should be inside (boundary included)")
	}
}

func TestWindowOvernight(t *testing.T) {
	w, _ := ParseWindow("23:00-01:00", "UTC")

	// 23:30 внутри
	t1 := time.Date(2026, 8, 17, 23, 30, 0, 0, time.UTC)
	if !w.Contains(t1) {
		t.Error("23:30 should be inside 23:00-01:00")
	}

	// 00:30 внутри
	t2 := time.Date(2026, 8, 18, 0, 30, 0, 0, time.UTC)
	if !w.Contains(t2) {
		t.Error("00:30 should be inside 23:00-01:00")
	}

	// 02:00 вне
	t3 := time.Date(2026, 8, 18, 2, 0, 0, 0, time.UTC)
	if w.Contains(t3) {
		t.Error("02:00 should be outside 23:00-01:00")
	}

	// 22:00 вне
	t4 := time.Date(2026, 8, 17, 22, 0, 0, 0, time.UTC)
	if w.Contains(t4) {
		t.Error("22:00 should be outside 23:00-01:00")
	}
}

func TestWindowAllDay(t *testing.T) {
	w, _ := ParseWindow("00:00-00:00", "UTC")

	// Любое время внутри
	for _, h := range []int{0, 6, 12, 18, 23} {
		tm := time.Date(2026, 8, 17, h, 0, 0, 0, time.UTC)
		if !w.Contains(tm) {
			t.Errorf("%02d:00 should be inside 00:00-00:00", h)
		}
	}
}

func TestWindowWithTimezone(t *testing.T) {
	// Окно 03:00-05:00 в Москве
	w, err := ParseWindow("03:00-05:00", "Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}

	// 04:00 МСК = 01:00 UTC
	moscow := time.Date(2026, 8, 17, 4, 0, 0, 0, w.Timezone)
	if !w.Contains(moscow) {
		t.Error("04:00 Moscow should be inside window")
	}
}
