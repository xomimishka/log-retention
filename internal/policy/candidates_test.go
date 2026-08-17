package policy

import (
	"testing"
	"time"

	"example/log-retention/internal/config"
	"example/log-retention/internal/fsmodel"
)

func makeFile(path string, modTime time.Time) fsmodel.FileInfo {
	return fsmodel.FileInfo{
		Path:    path,
		Size:    100,
		ModTime: modTime,
		IsDir:   false,
		Symlink: false,
	}
}

func TestSelectCandidatesIncludeExclude(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	snap := fsmodel.Snapshot{
		Version: 1,
		Now:     now,
		Roots:   []string{"/var/log"},
		Files: []fsmodel.FileInfo{
			makeFile("/var/log/app.log", now),
			makeFile("/var/log/app.log.1", now),
			makeFile("/var/log/app.tmp", now),
			makeFile("/var/log/sub/agent.log", now),
			{Path: "/var/log/dir", IsDir: true, ModTime: now},
			{Path: "/var/log/link.log", Symlink: true, ModTime: now},
		},
	}

	p := config.Policy{
		Name:      "p1",
		Roots:     []string{"/var/log"},
		Recursive: false,
		Include:   []string{"*.log"},
		Exclude:   []string{"*.tmp"},
	}

	got, err := SelectCandidates(snap, p)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{
		"/var/log/app.log":       true,
		"/var/log/sub/agent.log": true,
	}

	if len(got) != len(want) {
		t.Fatalf("got %d candidates, want %d", len(got), len(want))
	}
	for _, f := range got {
		if !want[f.Path] {
			t.Errorf("unexpected candidate: %s", f.Path)
		}
	}
}

func TestSelectCandidatesRecursive(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	snap := fsmodel.Snapshot{
		Version: 1,
		Now:     now,
		Roots:   []string{"/var/log"},
		Files: []fsmodel.FileInfo{
			makeFile("/var/log/app.log", now),
			makeFile("/var/log/sub/agent.log", now),
		},
	}

	p := config.Policy{
		Name:      "p1",
		Roots:     []string{"/var/log"},
		Recursive: true,
		Include:   []string{"sub/*.log"},
	}

	got, err := SelectCandidates(snap, p)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 {
		t.Fatalf("got %d candidates, want 1", len(got))
	}
	if got[0].Path != "/var/log/sub/agent.log" {
		t.Errorf("got %q, want /var/log/sub/agent.log", got[0].Path)
	}
}

func TestSelectCandidatesExcludesArchiveDest(t *testing.T) {
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	snap := fsmodel.Snapshot{
		Version: 1,
		Now:     now,
		Roots:   []string{"/var/log"},
		Files: []fsmodel.FileInfo{
			makeFile("/var/log/app.log", now),
			makeFile("/var/log/archive/app.zip", now),
		},
	}

	p := config.Policy{
		Name:  "p1",
		Roots: []string{"/var/log"},
		Archive: config.Archive{
			Dest: "/var/log/archive",
		},
	}

	got, err := SelectCandidates(snap, p)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 {
		t.Fatalf("got %d candidates, want 1", len(got))
	}
	if got[0].Path != "/var/log/app.log" {
		t.Errorf("got %q, want /var/log/app.log", got[0].Path)
	}
}
