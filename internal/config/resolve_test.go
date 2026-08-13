package config

import (
	"strings"
	"testing"
)

func TestResolveRelativeRoot(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Policies: []Policy{{
			Name:  "p1",
			Roots: []string{"relative/path"},
		}},
	}
	err := Resolve(cfg)
	if err == nil {
		t.Fatal("expected error for relative root")
	}
	if !strings.Contains(err.Error(), "not absolute") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResolveDotDotInRoot(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Policies: []Policy{{
			Name:  "p1",
			Roots: []string{"/var/log/../etc"},
		}},
	}
	err := Resolve(cfg)
	if err == nil {
		t.Fatal("expected error for .. in root")
	}
	if !strings.Contains(err.Error(), "contains ..") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResolveNestedRoots(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Policies: []Policy{{
			Name:  "p1",
			Roots: []string{"/var/log", "/var/log/app"},
		}},
	}
	err := Resolve(cfg)
	if err == nil {
		t.Fatal("expected error for nested roots")
	}
	if !strings.Contains(err.Error(), "nested") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResolveSameDestSameName(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Policies: []Policy{
			{Name: "p1", Roots: []string{"/var/log/a"}, Archive: Archive{Dest: "/var/log/archive", Name: "{group}.zip"}},
			{Name: "p2", Roots: []string{"/var/log/b"}, Archive: Archive{Dest: "/var/log/archive", Name: "{group}.zip"}},
		},
	}
	err := Resolve(cfg)
	if err == nil {
		t.Fatal("expected error for same dest and same name")
	}
	if !strings.Contains(err.Error(), "same archive.dest") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResolveSameDestDifferentName(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Policies: []Policy{
			{Name: "p1", Roots: []string{"/var/log/a"}, Archive: Archive{Dest: "/var/log/archive", Name: "{group}-a.zip"}},
			{Name: "p2", Roots: []string{"/var/log/b"}, Archive: Archive{Dest: "/var/log/archive", Name: "{group}-b.zip"}},
		},
	}
	err := Resolve(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveExpandsVariableInRoot(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Vars:    Vars{"LOGDIR": Var{Default: "/var/log"}},
		Policies: []Policy{{
			Name:  "p1",
			Roots: []string{"${LOGDIR}/app"},
		}},
	}
	t.Setenv("LOGDIR", "")

	err := Resolve(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Policies[0].Roots[0] != "/var/log/app" {
		t.Errorf("root = %q, want %q", cfg.Policies[0].Roots[0], "/var/log/app")
	}
}
