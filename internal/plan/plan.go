package plan

import (
	"encoding/json"
	"sort"
	"strings"
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

func (p *Plan) Normalize() {
	if p.PlanVersion == 0 {
		p.PlanVersion = PlanVersion
	}

	p.Now = p.Now.UTC()

	if p.Conflicts == nil {
		p.Conflicts = []Conflict{}
	}

	if p.Actions == nil {
		p.Actions = []Action{}
	}

	for i := range p.Conflicts {
		if p.Conflicts[i].Policies == nil {
			p.Conflicts[i].Policies = []string{}
		}
		sort.Strings(p.Conflicts[i].Policies)
	}

	sort.Slice(p.Conflicts, func(i, j int) bool {
		if p.Conflicts[i].Path != p.Conflicts[j].Path {
			return p.Conflicts[i].Path < p.Conflicts[j].Path
		}
		return p.Conflicts[i].Message < p.Conflicts[j].Message
	})

	sort.SliceStable(p.Actions, func(i, j int) bool {
		return lessAction(p.Actions[i], p.Actions[j])
	})

	for i := range p.Actions {
		if len(p.Actions[i].Reason.Facts) == 0 {
			p.Actions[i].Reason.Facts = nil
		}
	}
}

func MarshalPlanJSON(p Plan) ([]byte, error) {
	p.Normalize()

	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return nil, err
	}

	return append(b, '\n'), nil
}

func lessAction(a, b Action) bool {
	if a.Policy != b.Policy {
		return a.Policy < b.Policy
	}

	aRetention := isRetentionAction(a)
	bRetention := isRetentionAction(b)
	if aRetention != bRetention {
		return !aRetention
	}

	if a.Group != b.Group {
		return a.Group < b.Group
	}

	if a.Source != b.Source {
		return a.Source < b.Source
	}

	return actionRank(a.Kind) < actionRank(b.Kind)
}

// true, если действие это удаление архива
func isRetentionAction(a Action) bool {
	return a.Kind == KindDelete && strings.HasPrefix(a.Reason.Code, "retention_")
}

func actionRank(kind string) int {
	switch kind {
	case KindArchive:
		return 0
	case KindDelete:
		return 1
	default:
		return 2
	}
}
