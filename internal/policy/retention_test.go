package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"example/log-retention/internal/config"
	"example/log-retention/internal/plan"
)

func TestRetentionMaxAge(t *testing.T) {
	dir := t.TempDir()

	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	writeTestFile(t, dir, "app-2026-08-10.zip", "old")
	writeTestFile(t, dir, "app-2026-08-16.zip", "new")

	p := config.Policy{
		Name: "p1",
		Archive: config.Archive{
			Dest: filepath.ToSlash(dir),
			Name: "{group}-{date}.zip",
		},
		Retention: &config.Retention{
			MaxAgeDur: 5 * 24 * time.Hour,
		},
	}

	result, err := ApplyRetention(now, p, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.ToDelete) != 1 {
		t.Fatalf("expected 1 deletion, got %d", len(result.ToDelete))
	}
	if result.ToDelete[0].Reason.Code != plan.ReasonRetentionMaxAge {
		t.Errorf("code = %q, want %q", result.ToDelete[0].Reason.Code, plan.ReasonRetentionMaxAge)
	}
}

func TestRetentionMaxCount(t *testing.T) {
	dir := t.TempDir()

	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	writeTestFile(t, dir, "app-2026-08-15.zip", "1")
	writeTestFile(t, dir, "app-2026-08-16.zip", "2")
	writeTestFile(t, dir, "app-2026-08-17.zip", "3")

	p := config.Policy{
		Name: "p1",
		Archive: config.Archive{
			Dest: filepath.ToSlash(dir),
			Name: "{group}-{date}.zip",
		},
		Retention: &config.Retention{
			MaxCount: 2,
		},
	}

	result, err := ApplyRetention(now, p, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.ToDelete) != 1 {
		t.Fatalf("expected 1 deletion, got %d", len(result.ToDelete))
	}
	if !filepath.IsAbs(filepath.FromSlash(result.ToDelete[0].Source)) {
		t.Errorf("source path should be absolute: %q", result.ToDelete[0].Source)
	}
}

func TestRetentionKeepMinPriority(t *testing.T) {
	dir := t.TempDir()

	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	writeTestFile(t, dir, "app-2026-08-10.zip", "old")
	writeTestFile(t, dir, "app-2026-08-16.zip", "new")

	p := config.Policy{
		Name: "p1",
		Archive: config.Archive{
			Dest: filepath.ToSlash(dir),
			Name: "{group}-{date}.zip",
		},
		Retention: &config.Retention{
			MaxAgeDur: 5 * 24 * time.Hour,
			KeepMin:   2,
		},
	}

	result, err := ApplyRetention(now, p, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.ToDelete) != 0 {
		t.Fatalf("expected 0 deletions (keep_min priority), got %d", len(result.ToDelete))
	}
}

func TestRetentionForeignFile(t *testing.T) {
	dir := t.TempDir()

	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	writeTestFile(t, dir, "app-2026-08-16.zip", "archive")
	writeTestFile(t, dir, "readme.txt", "foreign")

	p := config.Policy{
		Name: "p1",
		Archive: config.Archive{
			Dest: filepath.ToSlash(dir),
			Name: "{group}-{date}.zip",
		},
		Retention: &config.Retention{
			MaxCount: 10,
		},
	}

	result, err := ApplyRetention(now, p, nil)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "foreign file") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected foreign file warning")
	}
}

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
