package policy

import (
	"testing"
	"time"

	"example/log-retention/internal/config"
	"example/log-retention/internal/fsmodel"
	"example/log-retention/internal/plan"
)

func makeFileAt(path string, modTime time.Time, size int64) fsmodel.FileInfo {
	return fsmodel.FileInfo{
		Path:    path,
		Size:    size,
		ModTime: modTime.UTC(),
		IsDir:   false,
		Symlink: false,
	}
}

func TestSelectKeepsGenerations(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)

	files := []fsmodel.FileInfo{
		makeFileAt("/var/log/app.log", now.Add(-1*time.Hour), 100),
		makeFileAt("/var/log/app.log.1", now.Add(-25*time.Hour), 100),
		makeFileAt("/var/log/app.log.2", now.Add(-49*time.Hour), 100),
		makeFileAt("/var/log/app.log.3", now.Add(-73*time.Hour), 100),
	}

	p := config.Policy{
		Name:  "p1",
		Group: config.Group{By: "name"},
		Select: config.Select{
			KeepGenerations: 2,
			MinAgeDur:       24 * time.Hour,
		},
		AfterArchive: "delete",
	}

	sel := SelectInGroup(now, p, "app", files)

	if len(sel.Actions) != 6 {
		t.Fatalf("got %d actions, want 6", len(sel.Actions))
	}

	if sel.Actions[0].Reason.Code != plan.ReasonKeptGeneration {
		t.Errorf("gen 1: got code %q, want %q", sel.Actions[0].Reason.Code, plan.ReasonKeptGeneration)
	}
	if sel.Actions[1].Reason.Code != plan.ReasonKeptGeneration {
		t.Errorf("gen 2: got code %q, want %q", sel.Actions[1].Reason.Code, plan.ReasonKeptGeneration)
	}

	if sel.Actions[2].Kind != plan.KindArchive {
		t.Errorf("gen 3: got kind %q, want %q", sel.Actions[2].Kind, plan.KindArchive)
	}
	if sel.Actions[3].Kind != plan.KindDelete {
		t.Errorf("gen 3 delete: got kind %q, want %q", sel.Actions[3].Kind, plan.KindDelete)
	}
	if sel.Actions[3].Reason.Code != plan.ReasonAfterArchiveDelete {
		t.Errorf("gen 3 delete: got code %q, want %q", sel.Actions[3].Reason.Code, plan.ReasonAfterArchiveDelete)
	}

	if sel.Actions[4].Kind != plan.KindArchive {
		t.Errorf("gen 4: got kind %q, want %q", sel.Actions[4].Kind, plan.KindArchive)
	}
	if sel.Actions[5].Kind != plan.KindDelete {
		t.Errorf("gen 4 delete: got kind %q, want %q", sel.Actions[5].Kind, plan.KindDelete)
	}
}

func TestSelectTooYoung(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	files := []fsmodel.FileInfo{
		makeFileAt("/var/log/app.log", now.Add(-1*time.Hour), 100),
	}

	p := config.Policy{
		Name:   "p1",
		Select: config.Select{MinAgeDur: 24 * time.Hour},
	}

	sel := SelectInGroup(now, p, "app", files)
	if sel.Actions[0].Reason.Code != plan.ReasonTooYoung {
		t.Errorf("got code %q, want %q", sel.Actions[0].Reason.Code, plan.ReasonTooYoung)
	}
}

func TestSelectTooSmall(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	files := []fsmodel.FileInfo{
		makeFileAt("/var/log/app.log", now.Add(-25*time.Hour), 100),
	}

	p := config.Policy{
		Name: "p1",
		Select: config.Select{
			MinAgeDur:  24 * time.Hour,
			MinSizeVal: 1024,
		},
	}

	sel := SelectInGroup(now, p, "app", files)
	if sel.Actions[0].Reason.Code != plan.ReasonTooSmall {
		t.Errorf("got code %q, want %q", sel.Actions[0].Reason.Code, plan.ReasonTooSmall)
	}
}

func TestSelectFutureMtime(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	files := []fsmodel.FileInfo{
		makeFileAt("/var/log/app.log", now.Add(24*time.Hour), 100),
	}

	p := config.Policy{
		Name:   "p1",
		Select: config.Select{MinAgeDur: 24 * time.Hour},
	}

	sel := SelectInGroup(now, p, "app", files)
	if sel.Actions[0].Reason.Code != plan.ReasonFutureMtime {
		t.Errorf("got code %q, want %q", sel.Actions[0].Reason.Code, plan.ReasonFutureMtime)
	}
	if sel.Actions[0].Kind != plan.KindSkip {
		t.Errorf("got kind %q, want %q", sel.Actions[0].Kind, plan.KindSkip)
	}
}

func TestSelectMinAgeZero(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	files := []fsmodel.FileInfo{
		makeFileAt("/var/log/app.log", now.Add(-1*time.Second), 100),
	}

	p := config.Policy{
		Name:   "p1",
		Select: config.Select{},
	}

	sel := SelectInGroup(now, p, "app", files)
	if sel.Actions[0].Kind != plan.KindArchive {
		t.Errorf("got kind %q, want %q", sel.Actions[0].Kind, plan.KindArchive)
	}
}

func TestSelectZeroSizeFileWithMinSizeZero(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	files := []fsmodel.FileInfo{
		makeFileAt("/var/log/empty.log", now.Add(-25*time.Hour), 0),
	}

	p := config.Policy{
		Name: "p1",
		Select: config.Select{
			MinAgeDur: 24 * time.Hour,
		},
	}

	sel := SelectInGroup(now, p, "empty", files)
	if sel.Actions[0].Kind != plan.KindArchive {
		t.Errorf("got kind %q, want %q", sel.Actions[0].Kind, plan.KindArchive)
	}
}

func TestSelectExactMinAgeBoundary(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	minAge := 24 * time.Hour
	files := []fsmodel.FileInfo{
		makeFileAt("/var/log/app.log", now.Add(-minAge), 100),
	}

	p := config.Policy{
		Name:   "p1",
		Select: config.Select{MinAgeDur: minAge},
	}

	sel := SelectInGroup(now, p, "app", files)
	if sel.Actions[0].Kind != plan.KindArchive {
		t.Errorf("exact boundary: got kind %q, want %q", sel.Actions[0].Kind, plan.KindArchive)
	}
}

func TestSelectExactMinSizeBoundary(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	files := []fsmodel.FileInfo{
		makeFileAt("/var/log/app.log", now.Add(-25*time.Hour), 1024),
	}

	p := config.Policy{
		Name: "p1",
		Select: config.Select{
			MinAgeDur:  24 * time.Hour,
			MinSizeVal: 1024,
		},
	}

	sel := SelectInGroup(now, p, "app", files)
	if sel.Actions[0].Kind != plan.KindArchive {
		t.Errorf("exact boundary: got kind %q, want %q", sel.Actions[0].Kind, plan.KindArchive)
	}
}

func TestSelectKeepGenerationsExceedsTotal(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	files := []fsmodel.FileInfo{
		makeFileAt("/var/log/app.log", now.Add(-1*time.Hour), 100),
		makeFileAt("/var/log/app.log.1", now.Add(-25*time.Hour), 100),
	}

	p := config.Policy{
		Name: "p1",
		Select: config.Select{
			KeepGenerations: 10,
			MinAgeDur:       24 * time.Hour,
		},
	}

	sel := SelectInGroup(now, p, "app", files)
	for i, a := range sel.Actions {
		if a.Reason.Code != plan.ReasonKeptGeneration {
			t.Errorf("action %d: got code %q, want %q", i, a.Reason.Code, plan.ReasonKeptGeneration)
		}
	}
}

func TestSelectTooYoungCheckedBeforeTooSmall(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	files := []fsmodel.FileInfo{
		makeFileAt("/var/log/app.log", now.Add(-1*time.Hour), 100),
	}

	p := config.Policy{
		Name: "p1",
		Select: config.Select{
			MinAgeDur:  24 * time.Hour,
			MinSizeVal: 1024,
		},
	}

	sel := SelectInGroup(now, p, "app", files)
	if sel.Actions[0].Reason.Code != plan.ReasonTooYoung {
		t.Errorf("got code %q, want %q (too_young must be reported first)",
			sel.Actions[0].Reason.Code, plan.ReasonTooYoung)
	}
}
