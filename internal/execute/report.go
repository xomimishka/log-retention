package execute

import (
	"encoding/json"
)

const (
	StatusDone    = "done"
	StatusSkipped = "skipped"
	StatusStale   = "stale"
	StatusFailed  = "failed"
	StatusLocked  = "locked"
)

// отчёт о прогоне
type Report struct {
	ReportVersion int            `json:"report_version"`
	Params        Params         `json:"params"`
	Totals        Totals         `json:"totals"`
	Warnings      []Warning      `json:"warnings"`
	Actions       []ActionResult `json:"actions"`
}

// параметры запуска
type Params struct {
	Config       string `json:"config"`
	Plan         string `json:"plan"`
	DryRun       bool   `json:"dry_run"`
	IgnoreWindow bool   `json:"ignore_window"`
	MaxDeletions int    `json:"max_deletions"`
}

// сводка по прогону
type Totals struct {
	Archived     int   `json:"archived"`
	Deleted      int   `json:"deleted"`
	Skipped      int   `json:"skipped"`
	Stale        int   `json:"stale"`
	Failed       int   `json:"failed"`
	BytesWritten int64 `json:"bytes_written"`
	BytesFreed   int64 `json:"bytes_freed"`
}

// нефатальное предупреждение
type Warning struct {
	Kind   string `json:"kind"`
	Target string `json:"target"`
}

// результат одного действия
type ActionResult struct {
	Kind   string `json:"kind"`
	Policy string `json:"policy"`
	Source string `json:"source"`
	Target string `json:"target,omitempty"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
	Error  string `json:"error,omitempty"`
}

func MarshalReportJSON(r Report) ([]byte, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
