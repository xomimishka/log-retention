package execute

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"example/log-retention/internal/config"
	"example/log-retention/internal/plan"
)

func makeTestFile(t *testing.T, dir, name, content string) (string, os.FileInfo) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.ToSlash(p), info
}

func testConfig(dest string) *config.Config {
	level := 9
	return &config.Config{
		Version: 1,
		Policies: []config.Policy{{
			Name:         "p1",
			Roots:        []string{"/tmp"},
			AfterArchive: "delete",
			Archive: config.Archive{
				Level: &level,
				Dest:  dest,
				Name:  "{group}-{date}.zip",
			},
		}},
	}
}

func TestExecuteArchiveAndDelete(t *testing.T) {
	dir := t.TempDir()
	srcPath, info := makeTestFile(t, dir, "app.log", "hello world")
	destDir := filepath.Join(dir, "archive")
	cfg := testConfig(filepath.ToSlash(destDir))

	target := filepath.ToSlash(filepath.Join(destDir, "app.zip"))
	now := time.Now().UTC()

	p := &plan.Plan{
		PlanVersion: plan.PlanVersion,
		Now:         now,
		Actions: []plan.Action{
			{
				Kind:    plan.KindArchive,
				Policy:  "p1",
				Source:  srcPath,
				Target:  target,
				Size:    info.Size(),
				ModTime: info.ModTime(),
			},
			{
				Kind:    plan.KindDelete,
				Policy:  "p1",
				Source:  srcPath,
				Size:    info.Size(),
				ModTime: info.ModTime(),
				Reason:  plan.Reason{Code: plan.ReasonAfterArchiveDelete},
			},
		},
	}

	opts := Options{MaxDeletions: 1000}
	report, err := Execute(context.Background(), p, cfg, opts)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if report.Totals.Archived != 1 {
		t.Errorf("Archived = %d, want 1", report.Totals.Archived)
	}
	if report.Totals.Deleted != 1 {
		t.Errorf("Deleted = %d, want 1", report.Totals.Deleted)
	}

	if _, err := os.Stat(filepath.FromSlash(srcPath)); !os.IsNotExist(err) {
		t.Error("source file should be deleted")
	}

	if _, err := os.Stat(filepath.FromSlash(target)); err != nil {
		t.Errorf("archive not created: %v", err)
	}
}

func TestExecuteDryRunDoesNotModifyDisk(t *testing.T) {
	dir := t.TempDir()
	srcPath, info := makeTestFile(t, dir, "app.log", "hello world")
	destDir := filepath.Join(dir, "archive")
	cfg := testConfig(filepath.ToSlash(destDir))

	target := filepath.ToSlash(filepath.Join(destDir, "app.zip"))
	now := time.Now().UTC()

	p := &plan.Plan{
		PlanVersion: plan.PlanVersion,
		Now:         now,
		Actions: []plan.Action{
			{
				Kind:    plan.KindArchive,
				Policy:  "p1",
				Source:  srcPath,
				Target:  target,
				Size:    info.Size(),
				ModTime: info.ModTime(),
			},
			{
				Kind:    plan.KindDelete,
				Policy:  "p1",
				Source:  srcPath,
				Size:    info.Size(),
				ModTime: info.ModTime(),
				Reason:  plan.Reason{Code: plan.ReasonAfterArchiveDelete},
			},
		},
	}

	opts := Options{DryRun: true, MaxDeletions: 1000}
	report, err := Execute(context.Background(), p, cfg, opts)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if report.Totals.Archived != 1 {
		t.Errorf("Archived = %d, want 1", report.Totals.Archived)
	}

	if _, err := os.Stat(filepath.FromSlash(srcPath)); err != nil {
		t.Error("source file should NOT be deleted in dry-run")
	}

	if _, err := os.Stat(filepath.FromSlash(target)); !os.IsNotExist(err) {
		t.Error("archive should NOT be created in dry-run")
	}
}

func TestExecuteStaleFile(t *testing.T) {
	dir := t.TempDir()
	srcPath, _ := makeTestFile(t, dir, "app.log", "hello world")
	destDir := filepath.Join(dir, "archive")
	cfg := testConfig(filepath.ToSlash(destDir))

	target := filepath.ToSlash(filepath.Join(destDir, "app.zip"))
	now := time.Now().UTC()

	p := &plan.Plan{
		PlanVersion: plan.PlanVersion,
		Now:         now,
		Actions: []plan.Action{
			{
				Kind:   plan.KindArchive,
				Policy: "p1",
				Source: srcPath,
				Target: target,
				Size:   99999,
			},
		},
	}

	opts := Options{MaxDeletions: 1000}
	report, err := Execute(context.Background(), p, cfg, opts)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if report.Totals.Stale != 1 {
		t.Errorf("Stale = %d, want 1", report.Totals.Stale)
	}
	if report.Actions[0].Status != StatusStale {
		t.Errorf("status = %q, want %q", report.Actions[0].Status, StatusStale)
	}
}

func TestExecuteMaxDeletionsExceeded(t *testing.T) {
	dir := t.TempDir()
	srcPath, info := makeTestFile(t, dir, "app.log", "hello")
	cfg := testConfig(filepath.ToSlash(filepath.Join(dir, "archive")))

	now := time.Now().UTC()
	p := &plan.Plan{
		PlanVersion: plan.PlanVersion,
		Now:         now,
		Actions: []plan.Action{
			{Kind: plan.KindDelete, Policy: "p1", Source: srcPath, Size: info.Size()},
			{Kind: plan.KindDelete, Policy: "p1", Source: srcPath, Size: info.Size()},
			{Kind: plan.KindDelete, Policy: "p1", Source: srcPath, Size: info.Size()},
		},
	}

	opts := Options{MaxDeletions: 2}
	_, err := Execute(context.Background(), p, cfg, opts)
	if err == nil {
		t.Fatal("expected error for exceeding max-deletions")
	}
}

func TestExecuteDeleteWithoutArchiveFails(t *testing.T) {
	dir := t.TempDir()
	srcPath, info := makeTestFile(t, dir, "app.log", "hello")
	cfg := testConfig(filepath.ToSlash(filepath.Join(dir, "archive")))

	now := time.Now().UTC()
	p := &plan.Plan{
		PlanVersion: plan.PlanVersion,
		Now:         now,
		Actions: []plan.Action{
			{
				Kind:    plan.KindDelete,
				Policy:  "p1",
				Source:  srcPath,
				Size:    info.Size(),
				ModTime: info.ModTime(),
				Reason:  plan.Reason{Code: plan.ReasonAfterArchiveDelete},
			},
		},
	}

	opts := Options{MaxDeletions: 1000}
	report, err := Execute(context.Background(), p, cfg, opts)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if report.Actions[0].Status != StatusFailed {
		t.Errorf("status = %q, want %q", report.Actions[0].Status, StatusFailed)
	}

	if _, err := os.Stat(filepath.FromSlash(srcPath)); err != nil {
		t.Error("source file should NOT be deleted without successful archive")
	}
}

func TestExecuteSymlinkNeverDeleted(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "real.log")
	if err := os.WriteFile(targetPath, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(dir, "link.log")
	if err := os.Symlink(targetPath, linkPath); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	cfg := testConfig(filepath.ToSlash(filepath.Join(dir, "archive")))
	linkSlash := filepath.ToSlash(linkPath)
	info, _ := os.Lstat(linkPath)

	now := time.Now().UTC()
	p := &plan.Plan{
		PlanVersion: plan.PlanVersion,
		Now:         now,
		Actions: []plan.Action{
			{
				Kind:   plan.KindDelete,
				Policy: "p1",
				Source: linkSlash,
				Size:   info.Size(),
			},
		},
	}

	opts := Options{MaxDeletions: 1000}
	report, err := Execute(context.Background(), p, cfg, opts)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if report.Actions[0].Status != StatusSkipped {
		t.Errorf("status = %q, want %q", report.Actions[0].Status, StatusSkipped)
	}
	if _, err := os.Lstat(linkPath); err != nil {
		t.Error("symlink should NOT be deleted")
	}
}

func TestExecuteCancellation(t *testing.T) {
	dir := t.TempDir()
	srcPath, info := makeTestFile(t, dir, "app.log", "hello")
	cfg := testConfig(filepath.ToSlash(filepath.Join(dir, "archive")))

	now := time.Now().UTC()
	p := &plan.Plan{
		PlanVersion: plan.PlanVersion,
		Now:         now,
		Actions: []plan.Action{
			{Kind: plan.KindSkip, Policy: "p1", Source: srcPath, Size: info.Size()},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	opts := Options{MaxDeletions: 1000}
	_, err := Execute(ctx, p, cfg, opts)
	if err == nil {
		t.Fatal("expected cancellation error")
	}
}
