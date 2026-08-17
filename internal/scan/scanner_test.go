package scan

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"example/log-retention/internal/config"
)

func makeTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	writeFile(t, filepath.Join(root, "app.log"), "hello")
	writeFile(t, filepath.Join(root, "app.log.1"), "old1")
	writeFile(t, filepath.Join(root, "app.log.2"), "old2")

	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(sub, "agent.log"), "agent")

	archive := filepath.Join(root, "archive")
	if err := os.Mkdir(archive, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(archive, "app-2026-08-10.zip"), "zipdata")

	fixedTime := time.Date(2026, 8, 17, 10, 0, 0, 0, time.UTC)
	fixTimes(t, root, fixedTime)

	return root
}

func fixTimes(t *testing.T, root string, tm time.Time) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return os.Chtimes(path, tm, tm)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanBasic(t *testing.T) {
	root := makeTree(t)

	cfg := &config.Config{
		Version: 1,
		Policies: []config.Policy{{
			Name:  "p1",
			Roots: []string{root},
			Archive: config.Archive{
				Dest: filepath.ToSlash(filepath.Join(root, "archive")),
			},
		}},
	}

	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	res, err := Scan(now, cfg)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	for _, f := range res.Snapshot.Files {
		if strings.Contains(f.Path, "/archive/") || strings.HasSuffix(f.Path, "/archive") {
			t.Errorf("archive.dest should be excluded, got %q", f.Path)
		}
	}

	found := map[string]bool{}
	for _, f := range res.Snapshot.Files {
		found[filepath.Base(f.Path)] = true
	}
	for _, want := range []string{"app.log", "app.log.1", "app.log.2", "agent.log"} {
		if !found[want] {
			t.Errorf("expected file %q in snapshot", want)
		}
	}

	if res.Snapshot.Now != now.UTC() {
		t.Errorf("Now = %v, want %v", res.Snapshot.Now, now.UTC())
	}

	for _, r := range res.Snapshot.Roots {
		if strings.ContainsRune(r, '\\') {
			t.Errorf("root %q contains backslash", r)
		}
	}
}

func TestScanDeterministic(t *testing.T) {
	root := makeTree(t)
	cfg := &config.Config{
		Version:  1,
		Policies: []config.Policy{{Name: "p1", Roots: []string{root}}},
	}

	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	r1, err := Scan(now, cfg)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := Scan(now, cfg)
	if err != nil {
		t.Fatal(err)
	}

	b1 := marshalJSON(t, r1.Snapshot)
	b2 := marshalJSON(t, r2.Snapshot)

	if string(b1) != string(b2) {
		t.Fatalf("scan is not deterministic\n\nfirst:\n%s\nsecond:\n%s", b1, b2)
	}
}

func marshalJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return b
}
