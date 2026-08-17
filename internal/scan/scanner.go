package scan

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"example/log-retention/internal/config"
	"example/log-retention/internal/fsmodel"
)

type Warning struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

func (w Warning) Error() string {
	return fmt.Sprintf("%s: %s", w.Path, w.Message)
}

type Result struct {
	Snapshot fsmodel.Snapshot
	Warnings []Warning
}

func Scan(now time.Time, cfg *config.Config) (*Result, error) {
	roots := collectRoots(cfg)
	exclude := collectArchiveDests(cfg)

	result := &Result{
		Snapshot: fsmodel.Snapshot{
			Version: fsmodel.SnapshotVersion,
			Now:     now.UTC(),
			Roots:   roots,
			Files:   []fsmodel.FileInfo{},
		},
	}

	for _, root := range roots {
		if err := scanRoot(filepath.FromSlash(root), exclude, result); err != nil {
			return nil, err
		}
	}

	result.Snapshot.Normalize()
	return result, nil
}

func collectRoots(cfg *config.Config) []string {
	seen := make(map[string]bool)
	var roots []string
	for _, p := range cfg.Policies {
		for _, r := range p.Roots {
			clean := toSlash(r)
			if !seen[clean] {
				seen[clean] = true
				roots = append(roots, clean)
			}
		}
	}
	sort.Strings(roots)
	return roots
}

func collectArchiveDests(cfg *config.Config) map[string]bool {
	dests := make(map[string]bool)
	for _, p := range cfg.Policies {
		if p.Archive.Dest != "" {
			dests[toSlash(p.Archive.Dest)] = true
		}
	}
	return dests
}

func scanRoot(root string, exclude map[string]bool, result *Result) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				result.Warnings = append(result.Warnings, Warning{
					Path:    toSlash(path),
					Message: "does not exist",
				})
				if d != nil && d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			result.Warnings = append(result.Warnings, Warning{
				Path:    toSlash(path),
				Message: err.Error(),
			})
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		slashPath := toSlash(path)

		if isExcluded(slashPath, exclude) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			result.Warnings = append(result.Warnings, Warning{
				Path:    slashPath,
				Message: err.Error(),
			})
			return nil
		}

		isSymlink := d.Type()&fs.ModeSymlink != 0

		result.Snapshot.Files = append(result.Snapshot.Files, fsmodel.FileInfo{
			Path:    slashPath,
			Size:    info.Size(),
			ModTime: info.ModTime().UTC(),
			IsDir:   info.IsDir(),
			Symlink: isSymlink,
		})

		return nil
	})
}

func isExcluded(slashPath string, exclude map[string]bool) bool {
	for dest := range exclude {
		if slashPath == dest || strings.HasPrefix(slashPath, dest+"/") {
			return true
		}
	}
	return false
}

func toSlash(p string) string {
	return filepath.ToSlash(filepath.Clean(p))
}
