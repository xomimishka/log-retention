package config

import (
	"strings"
	"testing"
)

func basePolicy() Policy {
	return Policy{
		Name:  "p1",
		Roots: []string{"/var/log"},
	}
}

func resolveOne(t *testing.T, p Policy) error {
	t.Helper()
	cfg := &Config{Version: 1, Policies: []Policy{p}}
	return Resolve(cfg)
}

func TestResolveInvalidIncludeMask(t *testing.T) {
	p := basePolicy()
	p.Include = []string{"["}
	err := resolveOne(t, p)
	if err == nil {
		t.Fatal("expected error for invalid mask")
	}
	if !strings.Contains(err.Error(), "invalid mask") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResolveValidMask(t *testing.T) {
	p := basePolicy()
	p.Include = []string{"*.log"}
	if err := resolveOne(t, p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveRegexpWithoutGroup(t *testing.T) {
	p := basePolicy()
	p.Group.By = "regexp"
	p.Group.Regexp = `^(\w+)\.log$`
	err := resolveOne(t, p)
	if err == nil {
		t.Fatal("expected error for regexp without named group")
	}
	if !strings.Contains(err.Error(), "named group") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResolveRegexpWithGroup(t *testing.T) {
	p := basePolicy()
	p.Group.By = "regexp"
	p.Group.Regexp = `^(?P<group>.+?)\.log$`
	if err := resolveOne(t, p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveInvalidWindow(t *testing.T) {
	p := basePolicy()
	p.Schedule.Window = "25:00-26:00"
	err := resolveOne(t, p)
	if err == nil {
		t.Fatal("expected error for invalid window")
	}
}

func TestResolveValidWindow(t *testing.T) {
	p := basePolicy()
	p.Schedule.Window = "03:00-05:00"
	if err := resolveOne(t, p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveMidnightWindow(t *testing.T) {
	p := basePolicy()
	p.Schedule.Window = "23:00-01:00"
	if err := resolveOne(t, p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveFullDayWindow(t *testing.T) {
	p := basePolicy()
	p.Schedule.Window = "00:00-00:00"
	if err := resolveOne(t, p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveUnknownTimezone(t *testing.T) {
	p := basePolicy()
	p.Schedule.Timezone = "Invalid/Zone"
	err := resolveOne(t, p)
	if err == nil {
		t.Fatal("expected error for unknown timezone")
	}
}

func TestResolveValidTimezone(t *testing.T) {
	p := basePolicy()
	p.Schedule.Timezone = "Europe/Moscow"
	if err := resolveOne(t, p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveUnknownSubstitution(t *testing.T) {
	p := basePolicy()
	p.Archive.Name = "{bogus}.zip"
	err := resolveOne(t, p)
	if err == nil {
		t.Fatal("expected error for unknown substitution")
	}
	if !strings.Contains(err.Error(), "unknown substitution") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResolveValidArchiveName(t *testing.T) {
	p := basePolicy()
	p.Archive.Name = "{group}-{date}.zip"
	if err := resolveOne(t, p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveInvalidUnit(t *testing.T) {
	p := basePolicy()
	p.Select.MinAge = "24hours"
	err := resolveOne(t, p)
	if err == nil {
		t.Fatal("expected error for invalid duration")
	}
}

func TestResolveParsesUnits(t *testing.T) {
	p := basePolicy()
	p.Select.MinAge = "24h"
	p.Select.MinSize = "1MiB"

	cfg := &Config{Version: 1, Policies: []Policy{p}}
	if err := Resolve(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := cfg.Policies[0]
	if got.Select.MinAgeDur.Hours() != 24 {
		t.Errorf("MinAgeDur = %v, want 24h", got.Select.MinAgeDur)
	}
	if got.Select.MinSizeVal != 1024*1024 {
		t.Errorf("MinSizeVal = %d, want %d", got.Select.MinSizeVal, 1024*1024)
	}
}
