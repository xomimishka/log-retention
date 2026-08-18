package policy

import (
	"time"

	"example/log-retention/internal/config"
	"example/log-retention/internal/fsmodel"
	"example/log-retention/internal/plan"
)

func BuildPlan(now time.Time, snap fsmodel.Snapshot, cfg *config.Config) (*plan.Plan, error) {
	result := &plan.Plan{
		PlanVersion: plan.PlanVersion,
		Now:         now.UTC(),
		Totals:      plan.Totals{},
		Conflicts:   []plan.Conflict{},
		Actions:     []plan.Action{},
	}

	filePolicyMap, err := BuildFilePolicyMap(snap, cfg.Policies)
	if err != nil {
		return nil, err
	}

	winners, conflicts := ResolveConflicts(filePolicyMap)

	for _, c := range conflicts {
		result.Conflicts = append(result.Conflicts, plan.Conflict{
			Path:     c.Path,
			Message:  c.String(),
			Policies: c.Policies,
		})
	}

	for _, p := range cfg.Policies {
		outside, err := IsOutsideWindow(now, p)
		if err != nil {
			return nil, err
		}

		candidates, err := SelectCandidates(snap, p)
		if err != nil {
			return nil, err
		}

		var filtered []fsmodel.FileInfo
		for _, f := range candidates {
			if winner, ok := winners[f.Path]; ok && winner == p.Name {
				filtered = append(filtered, f)
			}
		}

		groupResult, err := GroupCandidates(filtered, p)
		if err != nil {
			return nil, err
		}

		for _, f := range groupResult.NoGroup {
			result.Actions = append(result.Actions, skipAction(p, "", f, plan.ReasonNoGroup,
				"file did not match group regexp", nil))
		}

		for groupName, files := range groupResult.Groups {
			if outside {
				for _, f := range files {
					result.Actions = append(result.Actions, skipAction(p, groupName, f,
						plan.ReasonOutsideWindow, "policy is outside its schedule window", nil))
				}
				continue
			}

			sel := SelectInGroup(now, p, groupName, files)
			result.Actions = append(result.Actions, sel.Actions...)
		}
	}

	for _, f := range snap.Files {
		if f.Symlink {
			result.Actions = append(result.Actions, plan.Action{
				Kind:   plan.KindSkip,
				Policy: "",
				Source: f.Path,
				Size:   f.Size,
				Reason: plan.Reason{
					Code:    plan.ReasonSymlinkSkipped,
					Message: "symbolic links are never candidates",
				},
			})
		}
	}

	for _, f := range snap.Files {
		if f.IsDir || f.Symlink {
			continue
		}
		if _, ok := filePolicyMap[f.Path]; !ok {
			result.Actions = append(result.Actions, plan.Action{
				Kind:   plan.KindSkip,
				Policy: "",
				Source: f.Path,
				Size:   f.Size,
				Reason: plan.Reason{
					Code:    plan.ReasonNoPolicy,
					Message: "file did not match any policy",
				},
			})
		}
	}

	computeTotals(result)

	result.Normalize()

	return result, nil
}

func computeTotals(p *plan.Plan) {
	for _, a := range p.Actions {
		switch a.Kind {
		case plan.KindArchive:
			p.Totals.Archive++
			p.Totals.BytesToArchive += a.Size
		case plan.KindDelete:
			p.Totals.Delete++
			p.Totals.BytesToFree += a.Size
		case plan.KindSkip:
			p.Totals.Skip++
		}
	}
}
