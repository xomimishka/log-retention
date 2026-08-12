package config

import "testing"

func TestApplyDefaults(t *testing.T) {
	cfg := &Config{
		Version:  1,
		Policies: []Policy{{Name: "p", Roots: []string{"/var/log"}}},
	}

	cfg.ApplyDefaults()

	p := cfg.Policies[0]

	if p.Group.By != DefaultGroupBy {
		t.Errorf("Group.By = %q, want %q", p.Group.By, DefaultGroupBy)
	}
	if p.AfterArchive != DefaultAfterArchive {
		t.Errorf("AfterArchive = %q, want %q", p.AfterArchive, DefaultAfterArchive)
	}
	if p.Archive.Compress != DefaultCompress {
		t.Errorf("Archive.Compress = %q, want %q", p.Archive.Compress, DefaultCompress)
	}
	if p.Archive.Level == nil || *p.Archive.Level != DefaultLevel {
		t.Errorf("Archive.Level = %v, want %d", p.Archive.Level, DefaultLevel)
	}
	if p.Archive.Name != DefaultArchiveName {
		t.Errorf("Archive.Name = %q, want %q", p.Archive.Name, DefaultArchiveName)
	}
	if p.Retention != nil {
		t.Errorf("Retention = %+v, want nil", p.Retention)
	}
}

func TestApplyDefaultsKeepsExplicitLevelZero(t *testing.T) {
	zero := 0
	cfg := &Config{
		Version: 1,
		Policies: []Policy{{
			Name:    "p",
			Roots:   []string{"/var/log"},
			Archive: Archive{Level: &zero},
		}},
	}

	cfg.ApplyDefaults()

	got := cfg.Policies[0].Archive.Level
	if got == nil || *got != 0 {
		t.Errorf("explicit level 0 must be preserved, got %v", got)
	}
}

func TestApplyDefaultsKeepsExplicitRetention(t *testing.T) {
	cfg := &Config{
		Version: 1,
		Policies: []Policy{{
			Name:      "p",
			Roots:     []string{"/var/log"},
			Retention: &Retention{MaxCount: 5},
		}},
	}

	cfg.ApplyDefaults()

	if cfg.Policies[0].Retention == nil {
		t.Fatal("Retention must not be dropped")
	}
	if cfg.Policies[0].Retention.MaxCount != 5 {
		t.Errorf("Retention.MaxCount = %d, want 5", cfg.Policies[0].Retention.MaxCount)
	}
}
