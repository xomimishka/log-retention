package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
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
		if err := loadYAMLStrict(data, cfg); err != nil {
			return nil, err
		}
	case ".json":
		if err := loadJSONStrict(data, cfg); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported config extension: %q", filepath.Ext(path))
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}
	if err := Resolve(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func loadYAMLStrict(data []byte, cfg *Config) error {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("parse yaml: %w", err)
	}

	errs := &Errors{}
	checkYAMLNode(&root, "", reflect.TypeOf(Config{}), errs)
	if errs.Len() > 0 {
		return errs.Err()
	}

	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		if err := root.Content[0].Decode(cfg); err != nil {
			return fmt.Errorf("decode yaml: %w", err)
		}
	}
	return nil
}

func loadJSONStrict(data []byte, cfg *Config) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(cfg); err != nil {
		return fmt.Errorf("parse json: %w", err)
	}
	return nil
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
