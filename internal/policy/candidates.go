package policy

import (
	"path/filepath"
	"strings"

	"example/log-retention/internal/config"
	"example/log-retention/internal/expand"
	"example/log-retention/internal/fsmodel"
)

func SelectCandidates(snap fsmodel.Snapshot, p config.Policy) ([]fsmodel.FileInfo, error) {
	var out []fsmodel.FileInfo

	for _, f := range snap.Files {
		if f.IsDir || f.Symlink {
			continue
		}

		root, ok := findRoot(f.Path, p.Roots)
		if !ok {
			continue
		}

		if p.Archive.Dest != "" && isUnder(f.Path, p.Archive.Dest) {
			continue
		}

		var matchPath string
		if p.Recursive {
			matchPath = strings.TrimPrefix(f.Path, root+"/")
		} else {
			matchPath = filepath.Base(f.Path)
		}

		if len(p.Include) > 0 {
			matched := false
			for _, m := range p.Include {
				ok, err := expand.Match(m, matchPath)
				if err != nil {
					return nil, err
				}
				if ok {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		excluded := false
		for _, m := range p.Exclude {
			ok, err := expand.Match(m, matchPath)
			if err != nil {
				return nil, err
			}
			if ok {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		out = append(out, f)
	}

	return out, nil
}

func findRoot(path string, roots []string) (string, bool) {
	for _, r := range roots {
		if path == r || strings.HasPrefix(path, r+"/") {
			return r, true
		}
	}
	return "", false
}

func isUnder(path, dir string) bool {
	return path == dir || strings.HasPrefix(path, dir+"/")
}
