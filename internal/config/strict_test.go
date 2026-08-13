package config

import (
	"errors"
	"strings"
	"testing"
)

func TestLoadYAMLUnknownField(t *testing.T) {
	p := writeTemp(t, "lrt.yaml", `
version: 1
policies:
  - name: p1
    roots: ["/var/log"]
    unknown_field: true
`)
	_, err := Load(p)
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "unknown_field") {
		t.Errorf("error should mention field name: %v", err)
	}
}

func TestLoadYAMLDuplicateKey(t *testing.T) {
	p := writeTemp(t, "lrt.yaml", `
version: 1
version: 1
policies:
  - name: p1
    roots: ["/var/log"]
`)
	_, err := Load(p)
	if err == nil {
		t.Fatal("expected error for duplicate key")
	}
	if !strings.Contains(err.Error(), "duplicate key") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadYAMLMultipleErrors(t *testing.T) {
	p := writeTemp(t, "lrt.yaml", `
version: 1
unknown_top: true
policies:
  - name: p1
    roots: ["/var/log"]
    bad_field: 1
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

func TestLoadYAMLUnknownNestedField(t *testing.T) {
	p := writeTemp(t, "lrt.yaml", `
version: 1
policies:
  - name: p1
    roots: ["/var/log"]
    select:
      min_age: 24h
      bogus: true
`)
	_, err := Load(p)
	if err == nil {
		t.Fatal("expected error for unknown nested field")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("error should mention nested field: %v", err)
	}
}

func TestLoadYAMLDuplicateNestedKey(t *testing.T) {
	p := writeTemp(t, "lrt.yaml", `
version: 1
policies:
  - name: p1
    roots: ["/var/log"]
    select:
      min_age: 24h
      min_age: 48h
`)
	_, err := Load(p)
	if err == nil {
		t.Fatal("expected error for duplicate nested key")
	}
	if !strings.Contains(err.Error(), "duplicate key") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadJSONUnknownField(t *testing.T) {
	p := writeTemp(t, "lrt.json", `{
  "version": 1,
  "policies": [{"name": "p1", "roots": ["/var/log"], "extra": true}]
}`)
	_, err := Load(p)
	if err == nil {
		t.Fatal("expected error for unknown JSON field")
	}
}
