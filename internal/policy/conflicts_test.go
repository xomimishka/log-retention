package policy

import (
	"testing"
	"time"

	"example/log-retention/internal/config"
	"example/log-retention/internal/fsmodel"
)

func TestBuildFilePolicyMapSinglePolicy(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	snap := fsmodel.Snapshot{
		Version: 1,
		Now:     now,
		Roots:   []string{"/var/log"},
		Files: []fsmodel.FileInfo{
			{Path: "/var/log/app.log", Size: 100, ModTime: now},
			{Path: "/var/log/agent.log", Size: 100, ModTime: now},
		},
	}

	policies := []config.Policy{{
		Name:  "p1",
		Roots: []string{"/var/log"},
	}}

	fileMap, err := BuildFilePolicyMap(snap, policies)
	if err != nil {
		t.Fatal(err)
	}

	if len(fileMap) != 2 {
		t.Fatalf("got %d files, want 2", len(fileMap))
	}
	if len(fileMap["/var/log/app.log"]) != 1 {
		t.Errorf("app.log: got %d policies, want 1", len(fileMap["/var/log/app.log"]))
	}
}

func TestResolveConflictsSinglePolicy(t *testing.T) {
	p := config.Policy{Name: "p1", Priority: 100}
	fileMap := map[string][]config.Policy{
		"/var/log/app.log": {p},
	}

	winners, conflicts := ResolveConflicts(fileMap)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %d", len(conflicts))
	}
	if winners["/var/log/app.log"] != "p1" {
		t.Errorf("winner = %q, want %q", winners["/var/log/app.log"], "p1")
	}
}

func TestResolveConflictsDifferentPriorities(t *testing.T) {
	p1 := config.Policy{Name: "high", Priority: 100}
	p2 := config.Policy{Name: "low", Priority: 10}

	fileMap := map[string][]config.Policy{
		"/var/log/app.log": {p1, p2},
	}

	winners, conflicts := ResolveConflicts(fileMap)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %d", len(conflicts))
	}
	if winners["/var/log/app.log"] != "high" {
		t.Errorf("winner = %q, want %q", winners["/var/log/app.log"], "high")
	}
}

func TestResolveConflictsEqualPriorities(t *testing.T) {
	p1 := config.Policy{Name: "service-logs", Priority: 100}
	p2 := config.Policy{Name: "all-logs", Priority: 100}

	fileMap := map[string][]config.Policy{
		"/var/log/app.log": {p1, p2},
	}

	winners, conflicts := ResolveConflicts(fileMap)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if _, ok := winners["/var/log/app.log"]; ok {
		t.Error("conflicting file should not have a winner")
	}
	if conflicts[0].Path != "/var/log/app.log" {
		t.Errorf("conflict path = %q, want %q", conflicts[0].Path, "/var/log/app.log")
	}
}

func TestResolveConflictsOrderIndependence(t *testing.T) {
	p1 := config.Policy{Name: "b-policy", Priority: 50}
	p2 := config.Policy{Name: "a-policy", Priority: 50}

	fileMap1 := map[string][]config.Policy{
		"/var/log/app.log": {p1, p2},
	}
	_, conflicts1 := ResolveConflicts(fileMap1)

	fileMap2 := map[string][]config.Policy{
		"/var/log/app.log": {p2, p1},
	}
	_, conflicts2 := ResolveConflicts(fileMap2)

	if len(conflicts1) != len(conflicts2) {
		t.Fatalf("conflict count differs: %d vs %d", len(conflicts1), len(conflicts2))
	}
	if conflicts1[0].Path != conflicts2[0].Path {
		t.Errorf("conflict path differs: %q vs %q", conflicts1[0].Path, conflicts2[0].Path)
	}
}

func TestResolveConflictsMultipleFiles(t *testing.T) {
	p1 := config.Policy{Name: "p1", Priority: 100}
	p2 := config.Policy{Name: "p2", Priority: 100}
	p3 := config.Policy{Name: "p3", Priority: 50}

	fileMap := map[string][]config.Policy{
		"/var/log/a.log": {p1, p2},
		"/var/log/b.log": {p1, p3},
		"/var/log/c.log": {p3},
	}

	winners, conflicts := ResolveConflicts(fileMap)

	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Path != "/var/log/a.log" {
		t.Errorf("conflict path = %q, want %q", conflicts[0].Path, "/var/log/a.log")
	}

	if winners["/var/log/b.log"] != "p1" {
		t.Errorf("b.log winner = %q, want %q", winners["/var/log/b.log"], "p1")
	}
	if winners["/var/log/c.log"] != "p3" {
		t.Errorf("c.log winner = %q, want %q", winners["/var/log/c.log"], "p3")
	}
}

func TestResolveConflictsSortedOutput(t *testing.T) {
	p1 := config.Policy{Name: "p1", Priority: 100}
	p2 := config.Policy{Name: "p2", Priority: 100}

	fileMap := map[string][]config.Policy{
		"/var/log/z.log": {p1, p2},
		"/var/log/a.log": {p1, p2},
		"/var/log/m.log": {p1, p2},
	}

	_, conflicts := ResolveConflicts(fileMap)
	if len(conflicts) != 3 {
		t.Fatalf("expected 3 conflicts, got %d", len(conflicts))
	}

	expected := []string{"/var/log/a.log", "/var/log/m.log", "/var/log/z.log"}
	for i, c := range conflicts {
		if c.Path != expected[i] {
			t.Errorf("conflict[%d].Path = %q, want %q", i, c.Path, expected[i])
		}
	}
}
