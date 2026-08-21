package policy

import (
	"testing"
	"time"

	"example/log-retention/internal/config"
)

func TestRenderArchiveTargetAllSubstitutions(t *testing.T) {
	now := time.Date(2026, 8, 21, 14, 30, 0, 0, time.UTC)
	p := config.Policy{
		Name: "service-logs",
		Archive: config.Archive{
			Dest: "/var/log/archive",
			Name: "{group}-{date}-{policy}-{host}.zip",
		},
	}

	target := RenderArchiveTarget(p, "app", now)

	if !contains(target, "app") {
		t.Errorf("target should contain group 'app': %s", target)
	}
	if !contains(target, "2026-08-21") {
		t.Errorf("target should contain date '2026-08-21': %s", target)
	}
	if !contains(target, "service-logs") {
		t.Errorf("target should contain policy name: %s", target)
	}
	if !hasSuffix(target, ".zip") {
		t.Errorf("target should end with .zip: %s", target)
	}
}

func TestRenderArchiveTargetWithTimezone(t *testing.T) {
	now := time.Date(2026, 8, 21, 23, 30, 0, 0, time.UTC)
	p := config.Policy{
		Name: "p1",
		Archive: config.Archive{
			Dest: "/archive",
			Name: "{group}-{date}.zip",
		},
		Schedule: config.Schedule{
			Timezone: "Europe/Moscow",
		},
	}

	target := RenderArchiveTarget(p, "app", now)

	if !contains(target, "2026-08-22") {
		t.Errorf("target should contain Moscow date '2026-08-22': %s", target)
	}
}

func TestRenderArchiveTargetSanitizesInvalidChars(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	p := config.Policy{
		Name: "p1",
		Archive: config.Archive{
			Dest: "/archive",
			Name: "{group}<test>.zip",
		},
	}

	target := RenderArchiveTarget(p, "app", now)

	if contains(target, "<") || contains(target, ">") {
		t.Errorf("target should not contain < or >: %s", target)
	}
}

func TestRenderArchiveTargetDefaults(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	p := config.Policy{
		Name: "p1",
	}

	target := RenderArchiveTarget(p, "app", now)

	if !contains(target, "/tmp/archive") {
		t.Errorf("target should contain default dest '/tmp/archive': %s", target)
	}
	if !contains(target, "app-2026-08-21.zip") {
		t.Errorf("target should contain default name pattern: %s", target)
	}
}

func TestRenderArchiveTargetUnixSlashes(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	p := config.Policy{
		Name: "p1",
		Archive: config.Archive{
			Dest: "/var/log/archive",
			Name: "{group}-{date}.zip",
		},
	}

	target := RenderArchiveTarget(p, "app", now)

	if contains(target, "\\") {
		t.Errorf("target should not contain backslashes: %s", target)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
