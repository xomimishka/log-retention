package archive

import (
	"strings"
	"testing"
	"time"
)

func TestResolveNameAllSubstitutions(t *testing.T) {
	date := time.Date(2026, 8, 19, 3, 0, 0, 0, time.UTC)
	got := ResolveName("{group}-{date}.zip", "service", date, "host01", "service-logs")
	want := "service-2026-08-19.zip"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveNameHostAndPolicy(t *testing.T) {
	date := time.Date(2026, 8, 19, 0, 0, 0, 0, time.UTC)
	got := ResolveName("{policy}-{host}-{date}.zip", "app", date, "server1", "app-logs")
	want := "app-logs-server1-2026-08-19.zip"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveNameNoSubstitutions(t *testing.T) {
	date := time.Date(2026, 8, 19, 0, 0, 0, 0, time.UTC)
	got := ResolveName("fixed-name.zip", "app", date, "host", "policy")
	want := "fixed-name.zip"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSanitizeFileNameInvalidChars(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal.log", "normal.log"},
		{"file<name>.log", "file_name_.log"},
		{"file:name.log", "file_name.log"},
		{"file\"name.log", "file_name.log"},
		{"file|name.log", "file_name.log"},
		{"file?name.log", "file_name.log"},
		{"file*name.log", "file_name.log"},
		{"", "_"},
	}
	for _, tt := range tests {
		got := SanitizeFileName(tt.input)
		if got != tt.want {
			t.Errorf("SanitizeFileName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSanitizeFileNameReservedNames(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"CON", "_CON"},
		{"con.log", "_con.log"},
		{"NUL", "_NUL"},
		{"PRN.txt", "_PRN.txt"},
		{"CONSOLE", "CONSOLE"},
	}
	for _, tt := range tests {
		got := SanitizeFileName(tt.input)
		if got != tt.want {
			t.Errorf("SanitizeFileName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveCollisionNoCollision(t *testing.T) {
	exists := func(path string) bool { return false }
	got := ResolveCollision("/dest", "app.zip", false, exists)
	want := "/dest/app.zip"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveCollisionMergeSameDay(t *testing.T) {
	exists := func(path string) bool {
		return path == "/dest/app.zip"
	}
	got := ResolveCollision("/dest", "app.zip", true, exists)
	want := "/dest/app.zip"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveCollisionSuffix(t *testing.T) {
	exists := func(path string) bool {
		return path == "/dest/app.zip" || path == "/dest/app-2.zip"
	}
	got := ResolveCollision("/dest", "app.zip", false, exists)
	want := "/dest/app-3.zip"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveCollisionDeterministic(t *testing.T) {
	exists := func(path string) bool {
		return path == "/dest/app.zip"
	}
	got1 := ResolveCollision("/dest", "app.zip", false, exists)
	got2 := ResolveCollision("/dest", "app.zip", false, exists)
	if got1 != got2 {
		t.Errorf("non-deterministic: %q != %q", got1, got2)
	}
}

func TestResolveEntryCollisionNoCollision(t *testing.T) {
	exists := func(name string) bool { return false }
	got := ResolveEntryCollision("app.log", exists)
	if got != "app.log" {
		t.Errorf("got %q, want %q", got, "app.log")
	}
}

func TestResolveEntryCollisionWithSuffix(t *testing.T) {
	existing := map[string]bool{
		"app.log":   true,
		"app-2.log": true,
	}
	exists := func(name string) bool { return existing[name] }
	got := ResolveEntryCollision("app.log", exists)
	if got != "app-3.log" {
		t.Errorf("got %q, want %q", got, "app-3.log")
	}
}

func TestResolveEntryCollisionWithFolder(t *testing.T) {
	existing := map[string]bool{
		"2026-08-19/app.log": true,
	}
	exists := func(name string) bool { return existing[name] }
	got := ResolveEntryCollision("2026-08-19/app.log", exists)
	if got != "2026-08-19/app-2.log" {
		t.Errorf("got %q, want %q", got, "2026-08-19/app-2.log")
	}
}

func TestSanitizePreservesExtension(t *testing.T) {
	got := SanitizeFileName("my<file>.log")
	if !strings.HasSuffix(got, ".log") {
		t.Errorf("extension lost: %q", got)
	}
}
