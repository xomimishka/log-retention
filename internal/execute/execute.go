package execute

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"example/log-retention/internal/archive"
	"example/log-retention/internal/config"
	"example/log-retention/internal/plan"
)

type Options struct {
	DryRun       bool
	MaxDeletions int
	ConfigPath   string
	PlanPath     string
	IgnoreWindow bool
}

func Execute(ctx context.Context, p *plan.Plan, cfg *config.Config, opts Options) (*Report, error) {
	report := &Report{
		ReportVersion: 1,
		Params: Params{
			Config:       opts.ConfigPath,
			Plan:         opts.PlanPath,
			DryRun:       opts.DryRun,
			IgnoreWindow: opts.IgnoreWindow,
			MaxDeletions: opts.MaxDeletions,
		},
		Warnings: []Warning{},
		Actions:  []ActionResult{},
	}

	deleteCount := countDeletions(p)
	if deleteCount > opts.MaxDeletions {
		return nil, fmt.Errorf(
			"plan requires %d deletions, exceeding max-deletions %d",
			deleteCount, opts.MaxDeletions,
		)
	}

	archived := make(map[string]bool)

	for _, action := range p.Actions {
		if err := ctx.Err(); err != nil {
			return report, fmt.Errorf("cancelled: %w", err)
		}

		var result ActionResult
		switch action.Kind {
		case plan.KindArchive:
			result = executeArchive(action, cfg, opts, archived)
		case plan.KindDelete:
			result = executeDelete(action, cfg, opts, archived)
		case plan.KindSkip:
			result = ActionResult{
				Kind:   action.Kind,
				Policy: action.Policy,
				Source: action.Source,
				Status: StatusSkipped,
				Reason: action.Reason.Code,
			}
		default:
			result = ActionResult{
				Kind:   action.Kind,
				Policy: action.Policy,
				Source: action.Source,
				Status: StatusFailed,
				Error:  fmt.Sprintf("unknown action kind %q", action.Kind),
			}
		}

		report.Actions = append(report.Actions, result)
		updateTotals(report, result, action)
	}

	return report, nil
}

func countDeletions(p *plan.Plan) int {
	count := 0
	for _, a := range p.Actions {
		if a.Kind == plan.KindDelete {
			count++
		}
	}
	return count
}

func executeArchive(action plan.Action, cfg *config.Config, opts Options, archived map[string]bool) ActionResult {
	_ = cfg

	result := ActionResult{
		Kind:   action.Kind,
		Policy: action.Policy,
		Source: action.Source,
		Target: action.Target,
	}

	if err := checkFileUnchanged(action.Source, action.Size, action.ModTime); err != nil {
		result.Status = StatusStale
		result.Error = err.Error()
		return result
	}

	if opts.DryRun {
		result.Status = StatusDone
		archived[action.Source] = true
		return result
	}

	policy := findPolicy(cfg, action.Policy)
	archiveOpts := archive.Options{
		Level:           defaultLevel(policy),
		MergeSameDay:    policy.Archive.MergeSameDay,
		FolderInArchive: policy.Archive.FolderInArchive,
	}

	destDir := filepath.Dir(filepath.FromSlash(action.Target))
	baseName := filepath.Base(filepath.FromSlash(action.Target))
	target := archive.ResolveCollision(destDir, baseName, archiveOpts.MergeSameDay, func(path string) bool {
		_, err := os.Stat(path)
		return err == nil
	})

	entryName := filepath.Base(filepath.FromSlash(action.Source))
	sources := []archive.SourceFile{{
		Path:      action.Source,
		EntryName: entryName,
		Size:      action.Size,
		ModTime:   action.ModTime,
	}}

	_, _, err := archive.ArchiveFiles(filepath.ToSlash(target), sources, archiveOpts)
	if err != nil {
		result.Status = StatusFailed
		result.Error = err.Error()
		return result
	}

	result.Status = StatusDone
	result.Target = filepath.ToSlash(target)
	archived[action.Source] = true
	return result
}

func executeDelete(action plan.Action, cfg *config.Config, opts Options, archived map[string]bool) ActionResult {
	_ = cfg
	result := ActionResult{
		Kind:   action.Kind,
		Policy: action.Policy,
		Source: action.Source,
	}

	info, err := os.Lstat(filepath.FromSlash(action.Source))
	if err != nil {
		if os.IsNotExist(err) {
			result.Status = StatusStale
			result.Error = "file disappeared"
			return result
		}
		result.Status = StatusFailed
		result.Error = err.Error()
		return result
	}
	if info.Mode()&os.ModeSymlink != 0 {
		result.Status = StatusSkipped
		result.Reason = plan.ReasonSymlinkSkipped
		result.Error = "symbolic links are never deleted"
		return result
	}

	if info.IsDir() {
		result.Status = StatusSkipped
		result.Error = "directories are never deleted"
		return result
	}

	if action.Reason.Code == plan.ReasonAfterArchiveDelete {
		if !archived[action.Source] {
			result.Status = StatusFailed
			result.Error = "preceding archive did not succeed"
			return result
		}
	}

	if err := checkFileUnchanged(action.Source, action.Size, action.ModTime); err != nil {
		result.Status = StatusStale
		result.Error = err.Error()
		return result
	}

	if opts.DryRun {
		result.Status = StatusDone
		return result
	}

	if err := os.Remove(filepath.FromSlash(action.Source)); err != nil {
		result.Status = StatusFailed
		result.Error = err.Error()
		return result
	}

	result.Status = StatusDone
	return result
}

func checkFileUnchanged(path string, expectedSize int64, expectedModTime time.Time) error {
	info, err := os.Stat(filepath.FromSlash(path))
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file disappeared")
		}
		return fmt.Errorf("stat: %w", err)
	}
	if info.Size() != expectedSize {
		return fmt.Errorf("size changed: got %d, want %d", info.Size(), expectedSize)
	}
	if !expectedModTime.IsZero() && !info.ModTime().UTC().Equal(expectedModTime.UTC()) {
		return fmt.Errorf("mod time changed: got %s, want %s",
			info.ModTime().UTC().Format(time.RFC3339),
			expectedModTime.UTC().Format(time.RFC3339))
	}
	return nil
}

func findPolicy(cfg *config.Config, name string) config.Policy {
	for _, p := range cfg.Policies {
		if p.Name == name {
			return p
		}
	}
	return config.Policy{}
}

func defaultLevel(p config.Policy) int {
	if p.Archive.Level != nil {
		return *p.Archive.Level
	}
	return 9
}

func updateTotals(report *Report, result ActionResult, action plan.Action) {
	switch result.Status {
	case StatusDone:
		switch action.Kind {
		case plan.KindArchive:
			report.Totals.Archived++
			report.Totals.BytesWritten += action.Size
		case plan.KindDelete:
			report.Totals.Deleted++
			report.Totals.BytesFreed += action.Size
		}
	case StatusSkipped:
		report.Totals.Skipped++
	case StatusStale:
		report.Totals.Stale++
	case StatusFailed, StatusLocked:
		report.Totals.Failed++
	}
}
