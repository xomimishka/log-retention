package policy

import (
	"testing"
	"time"

	"example/log-retention/internal/config"
	"example/log-retention/internal/fsmodel"
	"example/log-retention/internal/plan"
)

func TestExplainFileTooYoung(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	snap := fsmodel.Snapshot{
		Version: 1,
		Now:     now,
		Roots:   []string{"/var/log"},
		Files: []fsmodel.FileInfo{
			{Path: "/var/log/app.log", Size: 100, ModTime: now.Add(-1 * time.Hour)},
		},
	}

	cfg := &config.Config{
		Version: 1,
		Policies: []config.Policy{{
			Name:   "p1",
			Roots:  []string{"/var/log"},
			Select: config.Select{MinAgeDur: 24 * time.Hour},
		}},
	}

	result, err := ExplainFile(now, snap, cfg, "/var/log/app.log")
	if err != nil {
		t.Fatal(err)
	}

	if result.Decision.Kind != plan.KindSkip {
		t.Errorf("kind = %q, want %q", result.Decision.Kind, plan.KindSkip)
	}
	if result.Decision.Reason.Code != plan.ReasonTooYoung {
		t.Errorf("code = %q, want %q", result.Decision.Reason.Code, plan.ReasonTooYoung)
	}
}

func TestExplainFileNoPolicy(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	snap := fsmodel.Snapshot{
		Version: 1,
		Now:     now,
		Roots:   []string{"/var/log"},
		Files: []fsmodel.FileInfo{
			{Path: "/var/other/file.txt", Size: 100, ModTime: now.Add(-25 * time.Hour)},
		},
	}

	cfg := &config.Config{
		Version: 1,
		Policies: []config.Policy{{
			Name:  "p1",
			Roots: []string{"/var/log"},
		}},
	}

	result, err := ExplainFile(now, snap, cfg, "/var/other/file.txt")
	if err != nil {
		t.Fatal(err)
	}

	if result.Decision.Reason.Code != plan.ReasonNoPolicy {
		t.Errorf("code = %q, want %q", result.Decision.Reason.Code, plan.ReasonNoPolicy)
	}
}

func TestExplainFileArchive(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	snap := fsmodel.Snapshot{
		Version: 1,
		Now:     now,
		Roots:   []string{"/var/log"},
		Files: []fsmodel.FileInfo{
			{Path: "/var/log/app.log", Size: 100, ModTime: now.Add(-25 * time.Hour)},
		},
	}

	cfg := &config.Config{
		Version: 1,
		Policies: []config.Policy{{
			Name:   "p1",
			Roots:  []string{"/var/log"},
			Select: config.Select{MinAgeDur: 24 * time.Hour},
		}},
	}

	result, err := ExplainFile(now, snap, cfg, "/var/log/app.log")
	if err != nil {
		t.Fatal(err)
	}

	if result.Decision.Kind != plan.KindArchive {
		t.Errorf("kind = %q, want %q", result.Decision.Kind, plan.KindArchive)
	}
}

func TestExplainFileConflict(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	snap := fsmodel.Snapshot{
		Version: 1,
		Now:     now,
		Roots:   []string{"/var/log"},
		Files: []fsmodel.FileInfo{
			{Path: "/var/log/app.log", Size: 100, ModTime: now.Add(-25 * time.Hour)},
		},
	}

	cfg := &config.Config{
		Version: 1,
		Policies: []config.Policy{
			{Name: "p1", Priority: 100, Roots: []string{"/var/log"}},
			{Name: "p2", Priority: 100, Roots: []string{"/var/log"}},
		},
	}

	result, err := ExplainFile(now, snap, cfg, "/var/log/app.log")
	if err != nil {
		t.Fatal(err)
	}

	if result.Decision.Reason.Code != "conflict" {
		t.Errorf("code = %q, want %q", result.Decision.Reason.Code, "conflict")
	}
	if len(result.MatchedPolicies) != 2 {
		t.Errorf("expected 2 matched policies, got %d", len(result.MatchedPolicies))
	}
	for _, m := range result.MatchedPolicies {
		if m.Selected {
			t.Error("no policy should be selected on conflict")
		}
	}
}

func TestExplainFileNotFound(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	snap := fsmodel.Snapshot{Version: 1, Now: now}

	cfg := &config.Config{Version: 1}

	_, err := ExplainFile(now, snap, cfg, "/missing")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
