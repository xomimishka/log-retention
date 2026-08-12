package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Errors struct {
	Items []error
}

func (e *Errors) Add(err error) {
	if err != nil {
		e.Items = append(e.Items, err)
	}
}

func (e *Errors) Len() int { return len(e.Items) }

func (e *Errors) Err() error {
	if len(e.Items) == 0 {
		return nil
	}
	return e
}

func (e *Errors) Error() string {
	var b strings.Builder
	for i, err := range e.Items {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(err.Error())
	}
	return b.String()
}

func (e *Errors) Unwrap() []error { return e.Items }

var policyNameRE = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &Config{}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse yaml: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse json: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config extension: %q", filepath.Ext(path))
	}

	return cfg, validate(cfg)
}

func validate(cfg *Config) error {
	errs := &Errors{}

	if cfg.Version != 1 {
		errs.Add(fmt.Errorf("config: unsupported version %d, expected 1", cfg.Version))
	}

	if len(cfg.Policies) == 0 {
		errs.Add(errors.New("config: policies must not be empty"))
	}

	seen := make(map[string]int, len(cfg.Policies))
	for i := range cfg.Policies {
		p := &cfg.Policies[i]

		if p.Name == "" {
			errs.Add(fmt.Errorf("config: policies[%d].name is required", i))
			continue
		}
		if !policyNameRE.MatchString(p.Name) {
			errs.Add(fmt.Errorf(
				"config: policies[%d].name %q contains invalid characters (allowed: A-Z, a-z, 0-9, -, _)",
				i, p.Name,
			))
		}
		if prev, ok := seen[p.Name]; ok {
			errs.Add(fmt.Errorf(
				"config: policy name %q is duplicated (policies[%d] and policies[%d])",
				p.Name, prev, i,
			))
		}
		seen[p.Name] = i
	}

	return errs.Err()
}
