package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"example/log-retention/internal/expand"
)

func Resolve(cfg *Config) error {
	cfg.ApplyDefaults()

	defaults := make(map[string]string, len(cfg.Vars))
	for name, v := range cfg.Vars {
		defaults[name] = v.Default
	}

	errs := &Errors{}

	for i := range cfg.Policies {
		resolvePolicy(&cfg.Policies[i], defaults, errs)
	}

	validateCrossPolicy(cfg, errs)

	return errs.Err()
}

func resolvePolicy(p *Policy, defaults map[string]string, errs *Errors) {
	for i, r := range p.Roots {
		field := fmt.Sprintf("roots[%d]", i)
		expanded, err := expand.ExpandVars(r, p.Name, field, defaults)
		if err != nil {
			errs.Add(err)
			continue
		}
		if hasDotDot(expanded) {
			errs.Add(fmt.Errorf("policy %q %s: path %q contains ..", p.Name, field, expanded))
			continue
		}
		p.Roots[i] = filepath.ToSlash(filepath.Clean(expanded))
	}

	if p.Archive.Dest != "" {
		expanded, err := expand.ExpandVars(p.Archive.Dest, p.Name, "archive.dest", defaults)
		if err != nil {
			errs.Add(err)
		} else if hasDotDot(expanded) {
			errs.Add(fmt.Errorf("policy %q archive.dest: path %q contains ..", p.Name, expanded))
		} else {
			p.Archive.Dest = filepath.ToSlash(filepath.Clean(expanded))
		}
	}

	for i, m := range p.Include {
		expanded, err := expand.ExpandVars(m, p.Name, fmt.Sprintf("include[%d]", i), defaults)
		if err != nil {
			errs.Add(err)
			continue
		}
		p.Include[i] = expanded
	}
	for i, m := range p.Exclude {
		expanded, err := expand.ExpandVars(m, p.Name, fmt.Sprintf("exclude[%d]", i), defaults)
		if err != nil {
			errs.Add(err)
			continue
		}
		p.Exclude[i] = expanded
	}

	validatePolicyPaths(p, errs)
	validatePolicyFields(p, errs)
}

func validatePolicyPaths(p *Policy, errs *Errors) {
	for i, r := range p.Roots {
		field := fmt.Sprintf("roots[%d]", i)
		if !isAbsPath(r) {
			errs.Add(fmt.Errorf("policy %q %s: path %q is not absolute", p.Name, field, r))
		}
	}

	if p.Archive.Dest != "" {
		if !isAbsPath(p.Archive.Dest) {
			errs.Add(fmt.Errorf("policy %q archive.dest: path %q is not absolute", p.Name, p.Archive.Dest))
		}
	}

	for i := 0; i < len(p.Roots); i++ {
		for j := i + 1; j < len(p.Roots); j++ {
			if isNested(p.Roots[i], p.Roots[j]) {
				errs.Add(fmt.Errorf("policy %q: roots[%d] %q is nested inside roots[%d] %q",
					p.Name, j, p.Roots[j], i, p.Roots[i]))
			} else if isNested(p.Roots[j], p.Roots[i]) {
				errs.Add(fmt.Errorf("policy %q: roots[%d] %q is nested inside roots[%d] %q",
					p.Name, i, p.Roots[i], j, p.Roots[j]))
			}
		}
	}
}

func validateCrossPolicy(cfg *Config, errs *Errors) {
	type destKey struct {
		dest string
		name string
	}
	seen := make(map[destKey]string)

	for i := range cfg.Policies {
		p := &cfg.Policies[i]
		if p.Archive.Dest == "" {
			continue
		}
		key := destKey{dest: p.Archive.Dest, name: p.Archive.Name}
		if prev, ok := seen[key]; ok {
			errs.Add(fmt.Errorf(
				"policies %q and %q share the same archive.dest %q with the same archive.name %q",
				prev, p.Name, p.Archive.Dest, p.Archive.Name))
		}
		seen[key] = p.Name
	}
}

func hasDotDot(p string) bool {
	for _, part := range strings.Split(filepath.ToSlash(p), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

func isNested(parent, child string) bool {
	parent = filepath.ToSlash(filepath.Clean(parent))
	child = filepath.ToSlash(filepath.Clean(child))
	if parent == child {
		return true
	}
	return strings.HasPrefix(child, parent+"/")
}

func isAbsPath(p string) bool {
	p = filepath.ToSlash(p)
	if strings.HasPrefix(p, "/") {
		return true
	}
	if len(p) >= 3 && p[1] == ':' && p[2] == '/' {
		return true
	}
	if strings.HasPrefix(p, "//") {
		return true
	}
	return false
}
