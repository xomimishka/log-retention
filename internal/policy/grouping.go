package policy

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"example/log-retention/internal/config"
	"example/log-retention/internal/fsmodel"
)

type GroupResult struct {
	Groups  map[string][]fsmodel.FileInfo
	NoGroup []fsmodel.FileInfo
}

func GroupCandidates(files []fsmodel.FileInfo, p config.Policy) (*GroupResult, error) {
	result := &GroupResult{
		Groups:  make(map[string][]fsmodel.FileInfo),
		NoGroup: []fsmodel.FileInfo{},
	}

	var re *regexp.Regexp
	if p.Group.By == "regexp" {
		var err error
		re, err = regexp.Compile(p.Group.Regexp)
		if err != nil {
			return nil, fmt.Errorf("policy %q: invalid group.regexp: %w", p.Name, err)
		}
	}

	for _, f := range files {
		group, ok, err := groupFor(f.Path, p.Group.By, re)
		if err != nil {
			return nil, err
		}
		if !ok {
			result.NoGroup = append(result.NoGroup, f)
			continue
		}
		result.Groups[group] = append(result.Groups[group], f)
	}

	for g := range result.Groups {
		sort.SliceStable(result.Groups[g], func(i, j int) bool {
			a, b := result.Groups[g][i], result.Groups[g][j]
			if !a.ModTime.Equal(b.ModTime) {
				return a.ModTime.After(b.ModTime)
			}
			return a.Path < b.Path
		})
	}

	return result, nil
}

func groupFor(filePath, groupBy string, re *regexp.Regexp) (string, bool, error) {
	name := filepath.Base(filePath)

	switch groupBy {
	case "dir":
		return path.Dir(filePath), true, nil

	case "name":
		group := stripGenerationSuffix(name)
		group = strings.TrimSuffix(group, filepath.Ext(group))
		return group, true, nil

	case "regexp":
		m := re.FindStringSubmatch(name)
		if m == nil {
			return "", false, nil
		}
		for i, subName := range re.SubexpNames() {
			if subName == "group" {
				return m[i], true, nil
			}
		}
		return "", false, nil

	default:
		return "", false, fmt.Errorf("unknown group.by %q", groupBy)
	}
}

func stripGenerationSuffix(name string) string {
	ext := filepath.Ext(name)
	if ext == "" {
		return name
	}
	numPart := strings.TrimPrefix(ext, ".")
	if numPart == "" {
		return name
	}
	if isAllDigits(numPart) {
		return strings.TrimSuffix(name, ext)
	}
	return name
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
