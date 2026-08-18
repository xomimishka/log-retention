package policy

import (
	"testing"
	"time"

	"example/log-retention/internal/config"
	"example/log-retention/internal/fsmodel"
	"example/log-retention/internal/plan"
)

func TestBuildPlanBasic(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)

	snap := fsmodel.Snapshot{
		Version: 1,
		Now:     now,
		Roots:   []string{"/var/log"},
		Files: []fsmodel.FileInfo{
			{Path: "/var/log/app.log", Size: 100, ModTime: now.Add(-25 * time.Hour)},
			{Path: "/var/log/app.log.1", Size: 100, ModTime: now.Add(-49 * time.Hour)},
		},
	}

	cfg := &config.Config{
		Version: 1,
		Policies: []config.Policy{{
			Name:         "p1",
			Roots:        []string{"/var/log"},
			Group:        config.Group{By: "name"},
			Select:       config.Select{MinAgeDur: 24 * time.Hour},
			AfterArchive: "delete",
		}},
	}

	p, err := BuildPlan(now, snap, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Actions) != 4 {
		t.Fatalf("got %d actions, want 4", len(p.Actions))
	}

	if p.Totals.Archive != 2 {
		t.Errorf("Totals.Archive = %d, want 2", p.Totals.Archive)
	}
	if p.Totals.Delete != 2 {
		t.Errorf("Totals.Delete = %d, want 2", p.Totals.Delete)
	}
}

func TestBuildPlanSymlinkSkipped(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)

	snap := fsmodel.Snapshot{
		Version: 1,
		Now:     now,
		Roots:   []string{"/var/log"},
		Files: []fsmodel.FileInfo{
			{Path: "/var/log/app.log", Size: 100, ModTime: now.Add(-25 * time.Hour)},
			{Path: "/var/log/link.log", Size: 100, ModTime: now, Symlink: true},
		},
	}

	cfg := &config.Config{
		Version: 1,
		Policies: []config.Policy{{
			Name:  "p1",
			Roots: []string{"/var/log"},
			Group: config.Group{By: "name"},
		}},
	}

	p, err := BuildPlan(now, snap, cfg)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, a := range p.Actions {
		if a.Source == "/var/log/link.log" && a.Reason.Code == plan.ReasonSymlinkSkipped {
			found = true
		}
	}
	if !found {
		t.Error("symlink should be skipped with symlink_skipped reason")
	}
}

func TestBuildPlanNoPolicy(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)

	snap := fsmodel.Snapshot{
		Version: 1,
		Now:     now,
		Roots:   []string{"/var/log"},
		Files: []fsmodel.FileInfo{
			{Path: "/var/log/app.log", Size: 100, ModTime: now.Add(-25 * time.Hour)},
			{Path: "/var/other/file.txt", Size: 100, ModTime: now.Add(-25 * time.Hour)},
		},
	}

	cfg := &config.Config{
		Version: 1,
		Policies: []config.Policy{{
			Name:  "p1",
			Roots: []string{"/var/log"},
			Group: config.Group{By: "name"},
		}},
	}

	p, err := BuildPlan(now, snap, cfg)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, a := range p.Actions {
		if a.Source == "/var/other/file.txt" && a.Reason.Code == plan.ReasonNoPolicy {
			found = true
		}
	}
	if !found {
		t.Error("file outside all policies should be skipped with no_policy reason")
	}
}

func TestBuildPlanOutsideWindow(t *testing.T) {
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
			Name:  "p1",
			Roots: []string{"/var/log"},
			Group: config.Group{By: "name"},
			Schedule: config.Schedule{
				Window:   "03:00-05:00",
				Timezone: "UTC",
			},
		}},
	}

	p, err := BuildPlan(now, snap, cfg)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, a := range p.Actions {
		if a.Source == "/var/log/app.log" && a.Reason.Code == plan.ReasonOutsideWindow {
			found = true
		}
	}
	if !found {
		t.Error("file should be skipped with outside_window reason")
	}
}

func TestBuildPlanConflict(t *testing.T) {
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
			{Name: "p1", Priority: 100, Roots: []string{"/var/log"}, Group: config.Group{By: "name"}},
			{Name: "p2", Priority: 100, Roots: []string{"/var/log"}, Group: config.Group{By: "name"}},
		},
	}

	p, err := BuildPlan(now, snap, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(p.Conflicts))
	}

	for _, a := range p.Actions {
		if a.Source == "/var/log/app.log" && a.Policy != "" {
			t.Errorf("conflicting file should not have actions from policies, got policy %q", a.Policy)
		}
	}
}

func TestBuildPlanDeterministic(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)

	snap := fsmodel.Snapshot{
		Version: 1,
		Now:     now,
		Roots:   []string{"/var/log"},
		Files: []fsmodel.FileInfo{
			{Path: "/var/log/b.log", Size: 100, ModTime: now.Add(-25 * time.Hour)},
			{Path: "/var/log/a.log", Size: 100, ModTime: now.Add(-25 * time.Hour)},
		},
	}

	cfg := &config.Config{
		Version: 1,
		Policies: []config.Policy{{
			Name:  "p1",
			Roots: []string{"/var/log"},
			Group: config.Group{By: "name"},
		}},
	}

	p1, err := BuildPlan(now, snap, cfg)
	if err != nil {
		t.Fatal(err)
	}

	snap.Files[0], snap.Files[1] = snap.Files[1], snap.Files[0]

	p2, err := BuildPlan(now, snap, cfg)
	if err != nil {
		t.Fatal(err)
	}

	b1, err := plan.MarshalPlanJSON(*p1)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := plan.MarshalPlanJSON(*p2)
	if err != nil {
		t.Fatal(err)
	}

	if string(b1) != string(b2) {
		t.Fatalf("plan is not deterministic\n\nfirst:\n%s\nsecond:\n%s", b1, b2)
	}
}
