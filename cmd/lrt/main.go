package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example/log-retention/internal/atomicfile"
	"example/log-retention/internal/config"
	"example/log-retention/internal/execute"
	"example/log-retention/internal/fsmodel"
	"example/log-retention/internal/plan"
	"example/log-retention/internal/policy"
	"example/log-retention/internal/scan"
)

const version = "0.1.0"

type exitCodeError struct {
	code int
	err  error
}

func (e *exitCodeError) Error() string { return e.err.Error() }
func (e *exitCodeError) Unwrap() error { return e.err }

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(2)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "version", "--version":
		fmt.Println(version)
		return
	case "help", "-h", "--help":
		usage(os.Stdout)
		return
	case "scan":
		err = runScan(ctx, args)
	case "plan":
		err = runPlan(ctx, args)
	case "explain":
		err = runExplain(ctx, args)
	case "apply":
		err = runApply(ctx, args)
	case "verify":
		err = runVerify(ctx, args)
	case "stat":
		err = runStat(ctx, args)
	case "report":
		err = runReport(ctx, args)
	default:
		fmt.Fprintf(os.Stderr, "lrt: unknown command: %q\n\n", cmd)
		usage(os.Stderr)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "lrt %s: %v\n", cmd, err)
		code := 2
		var ece *exitCodeError
		if errors.As(err, &ece) {
			code = ece.code
		}
		os.Exit(code)
	}
}

func usage(w *os.File) {
	fmt.Fprintf(w, `lrt — log retention tool

Usage:
  lrt <command> [flags]

Commands:
  scan      scan roots and write snapshot
  plan      build plan from snapshot/config
  explain   explain decision for one file
  apply     apply a plan
  verify    verify archives
  stat      show effective config/statistics
  report    render report
  version   print version
`)
}

func runScan(ctx context.Context, args []string) error {
	_ = ctx
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	configPath := fs.String("config", "", "path to config file (required)")
	outPath := fs.String("out", "", "path to output snapshot JSON (required)")
	fs.Parse(args)

	if *configPath == "" {
		return fmt.Errorf("--config is required")
	}
	if *outPath == "" {
		return fmt.Errorf("--out is required")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := config.Resolve(cfg); err != nil {
		return fmt.Errorf("resolve config: %w", err)
	}

	now := time.Now().UTC()
	result, err := scan.Scan(now, cfg)
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	for _, w := range result.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w.Error())
	}

	data, err := fsmodel.MarshalSnapshotJSON(result.Snapshot)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}

	if err := atomicfile.WriteBytes(*outPath, data); err != nil {
		return fmt.Errorf("write snapshot: %w", err)
	}

	fmt.Fprintf(os.Stderr, "snapshot: %d files written to %s\n",
		len(result.Snapshot.Files), *outPath)
	return nil
}

func runPlan(ctx context.Context, args []string) error {
	_ = ctx
	fs := flag.NewFlagSet("plan", flag.ExitOnError)
	configPath := fs.String("config", "", "path to config file (required)")
	snapshotPath := fs.String("snapshot", "", "path to snapshot JSON (optional, scans if empty)")
	outPath := fs.String("out", "", "path to output plan JSON (stdout if empty)")
	nowStr := fs.String("now", "", "run time in RFC3339 (for reproducibility)")
	includeSkipped := fs.Bool("include-skipped", false, "include skip actions in output")
	exitCode := fs.Bool("exit-code", false, "return 1 if plan is non-empty")
	fs.Parse(args)

	if *configPath == "" {
		return fmt.Errorf("--config is required")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := config.Resolve(cfg); err != nil {
		return fmt.Errorf("resolve config: %w", err)
	}

	now := time.Now().UTC()
	if *nowStr != "" {
		now, err = time.Parse(time.RFC3339, *nowStr)
		if err != nil {
			return fmt.Errorf("invalid --now: %w", err)
		}
		now = now.UTC()
	}

	var snap fsmodel.Snapshot
	if *snapshotPath != "" {
		data, err := os.ReadFile(*snapshotPath)
		if err != nil {
			return fmt.Errorf("read snapshot: %w", err)
		}
		snap, err = fsmodel.ParseSnapshotJSON(data)
		if err != nil {
			return fmt.Errorf("parse snapshot: %w", err)
		}
	} else {
		result, err := scan.Scan(now, cfg)
		if err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		for _, w := range result.Warnings {
			fmt.Fprintf(os.Stderr, "warning: %s\n", w.Error())
		}
		snap = result.Snapshot
	}

	p, err := policy.BuildPlan(now, snap, cfg)
	if err != nil {
		return fmt.Errorf("build plan: %w", err)
	}

	p.Config = *configPath
	cfgData, err := os.ReadFile(*configPath)
	if err != nil {
		return fmt.Errorf("read config for hash: %w", err)
	}
	cfgSHA := sha256.Sum256(cfgData)
	p.ConfigSHA256 = hex.EncodeToString(cfgSHA[:])

	planNonEmpty := false
	for _, a := range p.Actions {
		if a.Kind == plan.KindArchive || a.Kind == plan.KindDelete {
			planNonEmpty = true
			break
		}
	}

	if !*includeSkipped {
		filtered := make([]plan.Action, 0, len(p.Actions))
		for _, a := range p.Actions {
			if a.Kind != plan.KindSkip {
				filtered = append(filtered, a)
			}
		}
		p.Actions = filtered
	}

	data, err := plan.MarshalPlanJSON(*p)
	if err != nil {
		return fmt.Errorf("marshal plan: %w", err)
	}

	if *outPath != "" {
		if err := atomicfile.WriteBytes(*outPath, data); err != nil {
			return fmt.Errorf("write plan: %w", err)
		}
		fmt.Fprintf(os.Stderr, "plan written to %s\n", *outPath)
	} else {
		os.Stdout.Write(data)
	}

	if len(p.Conflicts) > 0 {
		return &exitCodeError{code: 1, err: fmt.Errorf("%d conflicts detected", len(p.Conflicts))}
	}
	if *exitCode && planNonEmpty {
		return &exitCodeError{code: 1, err: fmt.Errorf("plan is non-empty")}
	}

	return nil
}

func runExplain(ctx context.Context, args []string) error {
	_ = ctx
	fs := flag.NewFlagSet("explain", flag.ExitOnError)
	configPath := fs.String("config", "", "path to config file (required)")
	snapshotPath := fs.String("snapshot", "", "path to snapshot JSON (optional)")
	filePath := fs.String("file", "", "path to file to explain (required)")
	nowStr := fs.String("now", "", "run time in RFC3339")
	fs.Parse(args)

	if *configPath == "" {
		return fmt.Errorf("--config is required")
	}
	if *filePath == "" {
		return fmt.Errorf("--file is required")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := config.Resolve(cfg); err != nil {
		return fmt.Errorf("resolve config: %w", err)
	}

	now := time.Now().UTC()
	if *nowStr != "" {
		now, err = time.Parse(time.RFC3339, *nowStr)
		if err != nil {
			return fmt.Errorf("invalid --now: %w", err)
		}
		now = now.UTC()
	}

	var snap fsmodel.Snapshot
	if *snapshotPath != "" {
		data, err := os.ReadFile(*snapshotPath)
		if err != nil {
			return fmt.Errorf("read snapshot: %w", err)
		}
		snap, err = fsmodel.ParseSnapshotJSON(data)
		if err != nil {
			return fmt.Errorf("parse snapshot: %w", err)
		}
	} else {
		result, err := scan.Scan(now, cfg)
		if err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		snap = result.Snapshot
	}

	result, err := policy.ExplainFile(now, snap, cfg, *filePath)
	if err != nil {
		return fmt.Errorf("explain: %w", err)
	}

	fmt.Fprintf(os.Stderr, "file: %s\n", result.File.Path)
	fmt.Fprintf(os.Stderr, "size: %d\n", result.File.Size)
	fmt.Fprintf(os.Stderr, "mod time: %s\n", result.File.ModTime.UTC().Format(time.RFC3339))
	fmt.Fprintf(os.Stderr, "age: %s\n", result.Age)

	if len(result.MatchedPolicies) > 0 {
		fmt.Fprintf(os.Stderr, "\nmatched policies:\n")
		for _, m := range result.MatchedPolicies {
			arrow := ""
			if m.Selected {
				arrow = " <- selected"
			}
			fmt.Fprintf(os.Stderr, "  %s (priority %d)%s\n", m.Policy.Name, m.Policy.Priority, arrow)
			if m.Selected && m.Group != "" {
				fmt.Fprintf(os.Stderr, "    group: %s\n", m.Group)
			}
		}
	}

	fmt.Fprintf(os.Stderr, "\ndecision: %s (%s)\n", result.Decision.Kind, result.Decision.Reason.Code)
	if result.Decision.Reason.Message != "" {
		fmt.Fprintf(os.Stderr, "  %s\n", result.Decision.Reason.Message)
	}

	return nil
}

func runApply(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("apply", flag.ExitOnError)
	planPath := fs.String("plan", "", "path to plan JSON (required)")
	outPath := fs.String("out", "", "path to output report JSON (stdout if empty)")
	dryRun := fs.Bool("dry-run", false, "perform checks but do not modify disk")
	maxDeletions := fs.Int("max-deletions", 1000, "maximum number of deletions per run")
	maxPlanAge := fs.Duration("max-plan-age", time.Hour, "maximum age of the plan")
	force := fs.Bool("force", false, "ignore plan age limit")
	ignoreWindow := fs.Bool("ignore-window", false, "ignore schedule window")
	fs.Parse(args)

	if *planPath == "" {
		return fmt.Errorf("--plan is required")
	}

	planData, err := os.ReadFile(*planPath)
	if err != nil {
		return fmt.Errorf("read plan: %w", err)
	}
	p, err := plan.ParsePlanJSON(planData)
	if err != nil {
		return fmt.Errorf("parse plan: %w", err)
	}

	if !*force {
		age := time.Since(p.Now)
		if age > *maxPlanAge {
			return &exitCodeError{code: 2, err: fmt.Errorf(
				"plan is too old: %s (max-plan-age %s), use --force to override",
				age, *maxPlanAge)}
		}
	}

	if p.Config == "" {
		return fmt.Errorf("plan does not reference a config file")
	}
	cfg, err := config.Load(p.Config)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := config.Resolve(cfg); err != nil {
		return fmt.Errorf("resolve config: %w", err)
	}

	if p.ConfigSHA256 != "" {
		cfgData, err := os.ReadFile(p.Config)
		if err != nil {
			return fmt.Errorf("read config for hash: %w", err)
		}
		cfgSHA := sha256.Sum256(cfgData)
		cfgSHAHex := hex.EncodeToString(cfgSHA[:])
		if cfgSHAHex != p.ConfigSHA256 {
			return &exitCodeError{code: 2, err: fmt.Errorf(
				"plan was built from a different config: SHA-256 mismatch")}
		}
	}

	opts := execute.Options{
		DryRun:       *dryRun,
		MaxDeletions: *maxDeletions,
		ConfigPath:   p.Config,
		PlanPath:     *planPath,
		IgnoreWindow: *ignoreWindow,
	}
	report, err := execute.Execute(ctx, &p, cfg, opts)
	if err != nil {
		return err
	}

	reportData, err := execute.MarshalReportJSON(*report)
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}

	if *outPath != "" {
		if err := atomicfile.WriteBytes(*outPath, reportData); err != nil {
			return fmt.Errorf("write report: %w", err)
		}
		fmt.Fprintf(os.Stderr, "report written to %s\n", *outPath)
	} else {
		os.Stdout.Write(reportData)
	}

	fmt.Fprintf(os.Stderr, "apply: %d archived, %d deleted, %d skipped, %d stale, %d failed\n",
		report.Totals.Archived, report.Totals.Deleted, report.Totals.Skipped,
		report.Totals.Stale, report.Totals.Failed)

	if report.Totals.Stale > 0 || report.Totals.Failed > 0 {
		return &exitCodeError{code: 1, err: fmt.Errorf(
			"%d stale, %d failed actions", report.Totals.Stale, report.Totals.Failed)}
	}

	return nil
}

func runVerify(ctx context.Context, args []string) error {
	_ = ctx
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	fs.String("config", "", "path to config file")
	fs.String("quarantine-dir", "", "path to quarantine directory")
	fs.Parse(args)
	return fmt.Errorf("command 'verify' is not implemented yet")
}

func runStat(ctx context.Context, args []string) error {
	_ = ctx
	fs := flag.NewFlagSet("stat", flag.ExitOnError)
	fs.String("config", "", "path to config file")
	fs.Parse(args)
	return fmt.Errorf("command 'stat' is not implemented yet")
}

func runReport(ctx context.Context, args []string) error {
	_ = ctx
	fs := flag.NewFlagSet("report", flag.ExitOnError)
	fs.String("input", "", "path to report JSON")
	fs.String("format", "html", "output format (html or json)")
	fs.String("out", "", "path to output report")
	fs.Parse(args)
	return fmt.Errorf("command 'report' is not implemented yet")
}
