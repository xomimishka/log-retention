package config

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"example/log-retention/internal/expand"
)

var allowedSubstitutions = map[string]bool{
	"group":  true,
	"date":   true,
	"host":   true,
	"policy": true,
}

var substitutionRE = regexp.MustCompile(`\{([^}]*)\}`)

func validatePolicyFields(p *Policy, errs *Errors) {
	validateUnits(p, errs)
	validateMasks(p, errs)
	validateGroupRegexp(p, errs)
	validateSchedule(p, errs)
	validateArchiveName(p, errs)
}

func validateUnits(p *Policy, errs *Errors) {
	if p.Select.MinAge != "" {
		d, err := ParseDuration(p.Select.MinAge)
		if err != nil {
			errs.Add(fmt.Errorf("policy %q select.min_age: %w", p.Name, err))
		} else {
			p.Select.MinAgeDur = d
		}
	}
	if p.Select.MinSize != "" {
		v, err := ParseSize(p.Select.MinSize)
		if err != nil {
			errs.Add(fmt.Errorf("policy %q select.min_size: %w", p.Name, err))
		} else {
			p.Select.MinSizeVal = v
		}
	}
	if p.Retention != nil {
		if p.Retention.MaxAge != "" {
			d, err := ParseDuration(p.Retention.MaxAge)
			if err != nil {
				errs.Add(fmt.Errorf("policy %q retention.max_age: %w", p.Name, err))
			} else {
				p.Retention.MaxAgeDur = d
			}
		}
		if p.Retention.MaxTotalSize != "" {
			v, err := ParseSize(p.Retention.MaxTotalSize)
			if err != nil {
				errs.Add(fmt.Errorf("policy %q retention.max_total_size: %w", p.Name, err))
			} else {
				p.Retention.MaxTotalSizeVal = v
			}
		}
	}
}

func validateMasks(p *Policy, errs *Errors) {
	for i, m := range p.Include {
		if err := expand.ValidateMask(m); err != nil {
			errs.Add(fmt.Errorf("policy %q include[%d]: invalid mask %q: %v", p.Name, i, m, err))
		}
	}
	for i, m := range p.Exclude {
		if err := expand.ValidateMask(m); err != nil {
			errs.Add(fmt.Errorf("policy %q exclude[%d]: invalid mask %q: %v", p.Name, i, m, err))
		}
	}
}

func validateGroupRegexp(p *Policy, errs *Errors) {
	if p.Group.By != "regexp" {
		return
	}
	if p.Group.Regexp == "" {
		errs.Add(fmt.Errorf("policy %q: group.by is regexp but group.regexp is empty", p.Name))
		return
	}
	re, err := regexp.Compile(p.Group.Regexp)
	if err != nil {
		errs.Add(fmt.Errorf("policy %q group.regexp: invalid regexp: %v", p.Name, err))
		return
	}
	for _, name := range re.SubexpNames() {
		if name == "group" {
			return
		}
	}
	errs.Add(fmt.Errorf("policy %q group.regexp: must contain named group \"group\"", p.Name))
}

func validateSchedule(p *Policy, errs *Errors) {
	if p.Schedule.Window != "" {
		if err := parseWindow(p.Schedule.Window); err != nil {
			errs.Add(fmt.Errorf("policy %q schedule.window: %v", p.Name, err))
		}
	}
	if p.Schedule.Timezone != "" {
		if _, err := time.LoadLocation(p.Schedule.Timezone); err != nil {
			errs.Add(fmt.Errorf("policy %q schedule.timezone: unknown timezone %q", p.Name, p.Schedule.Timezone))
		}
	}
}

func parseWindow(s string) error {
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return fmt.Errorf("expected format HH:MM-HH:MM, got %q", s)
	}
	for _, part := range parts {
		if err := parseHHMM(part); err != nil {
			return err
		}
	}
	return nil
}

func parseHHMM(s string) error {
	if len(s) != 5 || s[2] != ':' {
		return fmt.Errorf("invalid time %q, expected HH:MM", s)
	}
	hh, mm := s[:2], s[3:]
	if !isDigits(hh) || !isDigits(mm) {
		return fmt.Errorf("invalid time %q, expected HH:MM", s)
	}
	h := int(hh[0]-'0')*10 + int(hh[1]-'0')
	m := int(mm[0]-'0')*10 + int(mm[1]-'0')
	if h > 23 {
		return fmt.Errorf("invalid hour in %q", s)
	}
	if m > 59 {
		return fmt.Errorf("invalid minute in %q", s)
	}
	return nil
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func validateArchiveName(p *Policy, errs *Errors) {
	checkTemplate(p.Archive.Name, "archive.name", p.Name, errs)
	checkTemplate(p.Archive.FolderInArchive, "archive.folder_in_archive", p.Name, errs)
}

func checkTemplate(tpl, field, policy string, errs *Errors) {
	if tpl == "" {
		return
	}
	for _, m := range substitutionRE.FindAllStringSubmatch(tpl, -1) {
		name := m[1]
		if !allowedSubstitutions[name] {
			errs.Add(fmt.Errorf("policy %q %s: unknown substitution {%s}", policy, field, name))
		}
	}
}
