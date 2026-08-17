package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example/log-retention/internal/atomicfile"
	"example/log-retention/internal/config"
	"example/log-retention/internal/fsmodel"
	"example/log-retention/internal/scan"
)

const version = "0.1.0"

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
		os.Exit(2)
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
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("cancelled: %w", err)
	}

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
	fs.String("config", "", "path to config file")
	fs.String("snapshot", "", "path to snapshot JSON")
	fs.String("out", "", "path to output plan JSON")
	fs.Parse(args)
	return fmt.Errorf("command 'plan' is not implemented yet")
}

func runExplain(ctx context.Context, args []string) error {
	_ = ctx
	fs := flag.NewFlagSet("explain", flag.ExitOnError)
	fs.String("config", "", "path to config file")
	fs.String("snapshot", "", "path to snapshot JSON")
	fs.String("file", "", "path to file to explain")
	fs.Parse(args)
	return fmt.Errorf("command 'explain' is not implemented yet")
}

func runApply(ctx context.Context, args []string) error {
	_ = ctx
	fs := flag.NewFlagSet("apply", flag.ExitOnError)
	fs.String("plan", "", "path to plan JSON")
	fs.Bool("dry-run", false, "do not modify disk")
	fs.Parse(args)
	return fmt.Errorf("command 'apply' is not implemented yet")
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
