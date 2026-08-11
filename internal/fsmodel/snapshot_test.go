package fsmodel

import (
	"bytes"
	"testing"
	"time"
)

func TestMarshalSnapshotJSONDeterministic(t *testing.T) {
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)

	s1 := Snapshot{
		Version: SnapshotVersion,
		Now:     now,
		Roots:   []string{"/var/log/app"},
		Files: []FileInfo{
			{
				Path:    "/var/log/app/b.log",
				Size:    20,
				ModTime: now,
			},
			{
				Path:    "/var/log/app/a.log",
				Size:    10,
				ModTime: now,
			},
		},
	}

	s2 := Snapshot{
		Version: SnapshotVersion,
		Now:     now,
		Roots:   []string{"/var/log/app"},
		Files: []FileInfo{
			{
				Path:    "/var/log/app/a.log",
				Size:    10,
				ModTime: now,
			},
			{
				Path:    "/var/log/app/b.log",
				Size:    20,
				ModTime: now,
			},
		},
	}

	b1, err := MarshalSnapshotJSON(s1)
	if err != nil {
		t.Fatal(err)
	}

	b2, err := MarshalSnapshotJSON(s2)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(b1, b2) {
		t.Fatalf("snapshot JSON is not deterministic\n\nfirst:\n%s\nsecond:\n%s", b1, b2)
	}
}

func TestParseSnapshotJSONRejectsBadVersion(t *testing.T) {
	data := []byte(`{"version":99,"now":"2026-08-11T12:00:00Z","roots":[],"files":[]}`)

	_, err := ParseSnapshotJSON(data)
	if err == nil {
		t.Fatal("expected error for bad version")
	}
}
