package plan

import (
	"time"
)

// версия формата плана
const PlanVersion = 1

// Виды действий
const (
	KindArchive = "archive"
	KindDelete  = "delete"
	KindSkip    = "skip"
)

// коды причин
const (
	ReasonMinAgeReached       = "min_age_reached"
	ReasonGenerationOverLimit = "generation_over_limit"
	ReasonAfterArchiveDelete  = "after_archive_delete"
	ReasonRetentionMaxAge     = "retention_max_age"
	ReasonRetentionMaxCount   = "retention_max_count"
	ReasonRetentionTotalSize  = "retention_total_size"
	ReasonTooYoung            = "too_young"
	ReasonTooSmall            = "too_small"
	ReasonKeptGeneration      = "kept_generation"
	ReasonKeptMin             = "kept_min"
	ReasonOutsideWindow       = "outside_window"
	ReasonNoGroup             = "no_group"
	ReasonSymlinkSkipped      = "symlink_skipped"
	ReasonFutureMtime         = "future_mtime"
	ReasonExcluded            = "excluded"
	ReasonNoPolicy            = "no_policy"
)

// Reason объясняет, почему действие попало в план.
type Reason struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Facts   map[string]string `json:"facts,omitempty"`
}

// Action описывает одну операцию плана.
type Action struct {
	Kind   string `json:"kind"`
	Policy string `json:"policy"`
	Group  string `json:"group,omitempty"`
	Source string `json:"source"`
	Target string `json:"target,omitempty"`
	Size   int64  `json:"size"`
	Reason Reason `json:"reason"`
}

// Totals — сводка по плану.
type Totals struct {
	Archive        int   `json:"archive"`
	Delete         int   `json:"delete"`
	Skip           int   `json:"skip"`
	BytesToArchive int64 `json:"bytes_to_archive"`
	BytesToFree    int64 `json:"bytes_to_free"`
}

// Conflict описывает конфликт политик.
type Conflict struct {
	Path     string   `json:"path"`
	Message  string   `json:"message"`
	Policies []string `json:"policies"`
}

// Plan — упорядоченный список действий, построенный по снимку и конфигурации.
type Plan struct {
	PlanVersion  int        `json:"plan_version"`
	Now          time.Time  `json:"now"`
	Config       string     `json:"config"`
	ConfigSHA256 string     `json:"config_sha256,omitempty"`
	Totals       Totals     `json:"totals"`
	Conflicts    []Conflict `json:"conflicts"`
	Actions      []Action   `json:"actions"`
}