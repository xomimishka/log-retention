package policy

import (
	"fmt"
	"time"

	"example/log-retention/internal/config"
	"example/log-retention/internal/fsmodel"
	"example/log-retention/internal/plan"
)

// решение по конкретному файлу
type ExplainResult struct {
	File            fsmodel.FileInfo
	Age             time.Duration
	MatchedPolicies []MatchedPolicy
	Decision        Decision
}

// описывает политику, под которую попал файл
type MatchedPolicy struct {
	Policy   config.Policy
	Selected bool
	Group    string
	Reason   plan.Reason
}

// описывает итоговое решение
type Decision struct {
	Kind   string
	Reason plan.Reason
}

// объясняет решение по конкретному файлу
func ExplainFile(now time.Time, snap fsmodel.Snapshot, cfg *config.Config, filePath string) (*ExplainResult, error) {
	// Ищем файл в снимке.
	var target fsmodel.FileInfo
	found := false
	for _, f := range snap.Files {
		if f.Path == filePath {
			target = f
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("file %q not found in snapshot", filePath)
	}

	age := now.Sub(target.ModTime)

	// определяем, под какие политики попадает файл
	filePolicyMap, err := BuildFilePolicyMap(snap, cfg.Policies)
	if err != nil {
		return nil, err
	}

	policies := filePolicyMap[filePath]
	if len(policies) == 0 {
		// Файл не попал ни под одну политику
		return &ExplainResult{
			File: target,
			Age:  age,
			Decision: Decision{
				Kind: plan.KindSkip,
				Reason: plan.Reason{
					Code:    plan.ReasonNoPolicy,
					Message: "file did not match any policy",
				},
			},
		}, nil
	}

	winner, isConflict := selectWinner(policies)
	matched := make([]MatchedPolicy, 0, len(policies))
	for _, p := range policies {
		selected := !isConflict && p.Name == winner
		group := determineGroup(target, p)
		matched = append(matched, MatchedPolicy{
			Policy:   p,
			Selected: selected,
			Group:    group,
		})
	}

	if isConflict {
		return &ExplainResult{
			File:            target,
			Age:             age,
			MatchedPolicies: matched,
			Decision: Decision{
				Kind: plan.KindSkip,
				Reason: plan.Reason{
					Code:    "conflict",
					Message: "file matched multiple policies with equal priority",
				},
			},
		}, nil
	}

	winnerPolicy := findPolicy(cfg.Policies, winner)
	sel := applySelection(now, winnerPolicy, target, age)

	return &ExplainResult{
		File:            target,
		Age:             age,
		MatchedPolicies: matched,
		Decision:        sel,
	}, nil
}

// определяет имя группы для файла в политике
func determineGroup(f fsmodel.FileInfo, p config.Policy) string {
	if p.Group.By == "" {
		p.Group.By = "dir"
	}

	candidates, err := SelectCandidates(
		fsmodel.Snapshot{Files: []fsmodel.FileInfo{f}},
		p,
	)
	if err != nil || len(candidates) == 0 {
		return ""
	}

	groupResult, err := GroupCandidates(candidates, p)
	if err != nil || groupResult == nil {
		return ""
	}

	var best string
	for g := range groupResult.Groups {
		if best == "" || g < best {
			best = g
		}
	}
	return best
}

// применяет правила отбора к одному файлу и возвращает решение
func applySelection(now time.Time, p config.Policy, f fsmodel.FileInfo, age time.Duration) Decision {
	if outside, _ := IsOutsideWindow(now, p); outside {
		return Decision{
			Kind: plan.KindSkip,
			Reason: plan.Reason{
				Code:    plan.ReasonOutsideWindow,
				Message: fmt.Sprintf("policy is outside its schedule window %s", p.Schedule.Window),
			},
		}
	}

	if age < 0 {
		return Decision{
			Kind: plan.KindSkip,
			Reason: plan.Reason{
				Code:    plan.ReasonFutureMtime,
				Message: fmt.Sprintf("modification time %s is in the future", f.ModTime.UTC().Format(time.RFC3339)),
			},
		}
	}

	if p.Select.MinAgeDur > 0 && age < p.Select.MinAgeDur {
		return Decision{
			Kind: plan.KindSkip,
			Reason: plan.Reason{
				Code:    plan.ReasonTooYoung,
				Message: fmt.Sprintf("age %s is less than min_age %s", age, p.Select.MinAgeDur),
				Facts: map[string]string{
					"age":     age.String(),
					"min_age": p.Select.MinAgeDur.String(),
				},
			},
		}
	}

	if p.Select.MinSizeVal > 0 && f.Size < p.Select.MinSizeVal {
		return Decision{
			Kind: plan.KindSkip,
			Reason: plan.Reason{
				Code:    plan.ReasonTooSmall,
				Message: fmt.Sprintf("size %d is less than min_size %d", f.Size, p.Select.MinSizeVal),
				Facts: map[string]string{
					"size":     fmt.Sprintf("%d", f.Size),
					"min_size": fmt.Sprintf("%d", p.Select.MinSizeVal),
				},
			},
		}
	}

	if p.AfterArchive == "delete" {
		return Decision{
			Kind: plan.KindArchive,
			Reason: plan.Reason{
				Code:    plan.ReasonMinAgeReached,
				Message: fmt.Sprintf("archive with subsequent delete (age=%s, size=%d)", age, f.Size),
				Facts: map[string]string{
					"age":           age.String(),
					"size":          fmt.Sprintf("%d", f.Size),
					"after_archive": "delete",
				},
			},
		}
	}

	return Decision{
		Kind: plan.KindArchive,
		Reason: plan.Reason{
			Code:    plan.ReasonMinAgeReached,
			Message: fmt.Sprintf("file passed selection (age=%s, size=%d)", age, f.Size),
			Facts: map[string]string{
				"age":  age.String(),
				"size": fmt.Sprintf("%d", f.Size),
			},
		},
	}
}

func findPolicy(policies []config.Policy, name string) config.Policy {
	for _, p := range policies {
		if p.Name == name {
			return p
		}
	}
	return config.Policy{}
}
