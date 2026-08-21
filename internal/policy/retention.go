package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"example/log-retention/internal/config"
	"example/log-retention/internal/plan"
)

type ArchiveInfo struct {
	Path    string
	Size    int64
	ModTime time.Time
	Date    time.Time
}

type RetentionResult struct {
	ToDelete []plan.Action
	Warnings []string
}

func ApplyRetention(now time.Time, p config.Policy, newlyCreatedPaths map[string]bool) (*RetentionResult, error) {
	result := &RetentionResult{
		ToDelete: []plan.Action{},
		Warnings: []string{},
	}

	if p.Retention == nil {
		return result, nil
	}
	if p.Retention.MaxAgeDur == 0 && p.Retention.MaxCount == 0 && p.Retention.MaxTotalSizeVal == 0 {
		return result, nil
	}

	destDir := p.Archive.Dest
	if destDir == "" {
		return result, nil
	}

	entries, err := os.ReadDir(filepath.FromSlash(destDir))
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, fmt.Errorf("read archive dir %q: %w", destDir, err)
	}

	var archives []ArchiveInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("cannot stat %s: %v", entry.Name(), err))
			continue
		}

		matches, date := matchesArchiveName(entry.Name(), p)
		if !matches {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("foreign file in dest: %s", filepath.ToSlash(filepath.Join(destDir, entry.Name()))))
			continue
		}

		archives = append(archives, ArchiveInfo{
			Path:    filepath.ToSlash(filepath.Join(destDir, entry.Name())),
			Size:    info.Size(),
			ModTime: info.ModTime().UTC(),
			Date:    date,
		})
	}

	sort.SliceStable(archives, func(i, j int) bool {
		if !archives[i].Date.Equal(archives[j].Date) {
			return archives[i].Date.After(archives[j].Date)
		}
		if !archives[i].ModTime.Equal(archives[j].ModTime) {
			return archives[i].ModTime.After(archives[j].ModTime)
		}
		return archives[i].Path < archives[j].Path
	})

	toDelete := make(map[string]bool)

	if p.Retention.MaxAgeDur > 0 {
		for _, a := range archives {
			age := now.Sub(a.Date)
			if age > p.Retention.MaxAgeDur {
				toDelete[a.Path] = true
			}
		}
	}

	if p.Retention.MaxCount > 0 {
		kept := 0
		for _, a := range archives {
			if kept >= p.Retention.MaxCount {
				toDelete[a.Path] = true
			} else {
				kept++
			}
		}
	}

	if p.Retention.MaxTotalSizeVal > 0 {
		totalSize := int64(0)
		for _, a := range archives {
			totalSize += a.Size
		}

		for i := len(archives) - 1; i >= 0 && totalSize > p.Retention.MaxTotalSizeVal; i-- {
			toDelete[archives[i].Path] = true
			totalSize -= archives[i].Size
		}

		if len(archives) == 1 && archives[0].Size > p.Retention.MaxTotalSizeVal && p.Retention.KeepMin > 0 {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("archive %s exceeds max_total_size but kept_min > 0", archives[0].Path))
		}
	}

	if p.Retention.KeepMin > 0 {
		if p.Retention.MaxCount > 0 && p.Retention.KeepMin > p.Retention.MaxCount {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("keep_min (%d) exceeds max_count (%d), keep_min wins",
					p.Retention.KeepMin, p.Retention.MaxCount))
		}

		kept := 0
		for _, a := range archives {
			if kept < p.Retention.KeepMin {
				delete(toDelete, a.Path)
				kept++
			}
		}
	}

	for _, a := range archives {
		if !toDelete[a.Path] {
			continue
		}

		code, message := determineRetentionReason(now, a, p)

		result.ToDelete = append(result.ToDelete, plan.Action{
			Kind:   plan.KindDelete,
			Policy: p.Name,
			Source: a.Path,
			Size:   a.Size,
			Reason: plan.Reason{
				Code:    code,
				Message: message,
				Facts: map[string]string{
					"age":  now.Sub(a.Date).String(),
					"size": fmt.Sprintf("%d", a.Size),
				},
			},
		})
	}

	return result, nil
}

func matchesArchiveName(name string, p config.Policy) (bool, time.Time) {
	datePattern := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	matches := datePattern.FindStringSubmatch(name)
	if len(matches) < 2 {
		return false, time.Time{}
	}

	date, err := time.Parse("2006-01-02", matches[1])
	if err != nil {
		return false, time.Time{}
	}

	return true, date
}

func determineRetentionReason(now time.Time, a ArchiveInfo, p config.Policy) (string, string) {
	_ = p
	age := now.Sub(a.Date)

	if p.Retention != nil && p.Retention.MaxAgeDur > 0 && age > p.Retention.MaxAgeDur {
		return plan.ReasonRetentionMaxAge,
			fmt.Sprintf("archive age %s exceeds max_age %s", age, p.Retention.MaxAgeDur)
	}

	if p.Retention != nil && p.Retention.MaxCount > 0 {
		return plan.ReasonRetentionMaxCount,
			fmt.Sprintf("archive exceeds max_count %d", p.Retention.MaxCount)
	}

	if p.Retention != nil && p.Retention.MaxTotalSizeVal > 0 {
		return plan.ReasonRetentionTotalSize,
			fmt.Sprintf("archive contributes to total size exceeding %d", p.Retention.MaxTotalSizeVal)
	}

	return plan.ReasonRetentionMaxAge, "retention policy"
}
