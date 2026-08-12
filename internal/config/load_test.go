package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadYAML(t *testing.T) {
	p := writeTemp(t, "lrt.yaml", `
version: 1
policies:
  - name: service-logs
    roots: ["/var/log"]
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(cfg.Policies) != 1 {
		t.Fatalf("got %d policies, want 1", len(cfg.Policies))
	}
	if cfg.Policies[0].Name != "service-logs" {
		t.Errorf("name = %q, want %q", cfg.Policies[0].Name, "service-logs")
	}
}

func TestLoadJSON(t *testing.T) {
	p := writeTemp(t, "lrt.json", `{
  "version": 1,
  "policies": [{"name": "p1", "roots": ["/var/log"]}]
}`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(cfg.Policies) != 1 {
		t.Fatalf("got %d policies, want 1", len(cfg.Policies))
	}
}

func TestLoadUnsupportedExtension(t *testing.T) {
	p := writeTemp(t, "lrt.toml", "")
	_, err := Load(p)
	if err == nil {
		t.Fatal("expected error for unsupported extension")
	}
}

func TestLoadBadVersion(t *testing.T) {
	p := writeTemp(t, "lrt.yaml", `
version: 2
policies:
  - name: p1
    roots: ["/var/log"]
`)
	_, err := Load(p)
	if err == nil {
		t.Fatal("expected error for wrong version")
	}
	if !strings.Contains(err.Error(), "unsupported version 2") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadEmptyPolicies(t *testing.T) {
	p := writeTemp(t, "lrt.yaml", `
version: 1
policies: []
`)
	_, err := Load(p)
	if err == nil {
		t.Fatal("expected error for empty policies")
	}
	if !strings.Contains(err.Error(), "policies must not be empty") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadDuplicatePolicyNames(t *testing.T) {
	p := writeTemp(t, "lrt.yaml", `
version: 1
policies:
  - name: same
    roots: ["/var/log/a"]
  - name: same
    roots: ["/var/log/b"]
`)
	_, err := Load(p)
	if err == nil {
		t.Fatal("expected error for duplicate policy names")
	}
	if !strings.Contains(err.Error(), "duplicated") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadInvalidPolicyName(t *testing.T) {
	p := writeTemp(t, "lrt.yaml", `
version: 1
policies:
  - name: "bad name!"
    roots: ["/var/log"]
`)
	_, err := Load(p)
	if err == nil {
		t.Fatal("expected error for invalid policy name")
	}
	if !strings.Contains(err.Error(), "invalid characters") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadCollectsMultipleErrors(t *testing.T) {
	p := writeTemp(t, "lrt.yaml", `
version: 99
policies:
  - name: a
    roots: ["/var/log"]
  - name: a
    roots: ["/var/log"]
`)
	_, err := Load(p)
	if err == nil {
		t.Fatal("expected errors")
	}

	var errs *Errors
	if !errors.As(err, &errs) {
		t.Fatalf("expected *Errors, got %T: %v", err, err)
	}
	if len(errs.Items) < 2 {
		t.Errorf("expected at least 2 errors, got %d: %v", len(errs.Items), errs.Items)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("does-not-exist.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
