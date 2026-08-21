package policy

import (
	"fmt"
	"time"

	"example/log-retention/internal/config"
	"example/log-retention/internal/fsmodel"
	"example/log-retention/internal/plan"
)

type Selection struct {
	Actions []plan.Action
}

func SelectInGroup(now time.Time, p config.Policy, group string, files []fsmodel.FileInfo) Selection {
	var sel Selection

	for i, f := range files {
		gen := i + 1
		actions := decideFileActions(now, p, group, f, gen, len(files))
		sel.Actions = append(sel.Actions, actions...)
	}

	return sel
}

func decideFileActions(now time.Time, p config.Policy, group string, f fsmodel.FileInfo, gen, total int) []plan.Action {
	age := now.Sub(f.ModTime)

	if age < 0 {
		return []plan.Action{skipAction(p, group, f, plan.ReasonFutureMtime,
			fmt.Sprintf("modification time %s is in the future", f.ModTime.UTC().Format(time.RFC3339)),
			map[string]string{
				"mod_time": f.ModTime.UTC().Format(time.RFC3339),
				"now":      now.UTC().Format(time.RFC3339),
			})}
	}

	if p.Select.KeepGenerations > 0 && gen <= p.Select.KeepGenerations {
		return []plan.Action{skipAction(p, group, f, plan.ReasonKeptGeneration,
			fmt.Sprintf("generation %d of %d kept (keep_generations=%d)", gen, total, p.Select.KeepGenerations),
			map[string]string{
				"generation":        fmt.Sprintf("%d", gen),
				"total_generations": fmt.Sprintf("%d", total),
				"keep_generations":  fmt.Sprintf("%d", p.Select.KeepGenerations),
			})}
	}

	if p.Select.MinAgeDur > 0 && age < p.Select.MinAgeDur {
		return []plan.Action{skipAction(p, group, f, plan.ReasonTooYoung,
			fmt.Sprintf("age %s is less than min_age %s", age, p.Select.MinAgeDur),
			map[string]string{
				"age":     age.String(),
				"min_age": p.Select.MinAgeDur.String(),
			})}
	}

	if p.Select.MinSizeVal > 0 && f.Size < p.Select.MinSizeVal {
		return []plan.Action{skipAction(p, group, f, plan.ReasonTooSmall,
			fmt.Sprintf("size %d is less than min_size %d", f.Size, p.Select.MinSizeVal),
			map[string]string{
				"size":     fmt.Sprintf("%d", f.Size),
				"min_size": fmt.Sprintf("%d", p.Select.MinSizeVal),
			})}
	}

	actions := []plan.Action{archiveAction(now, p, group, f, age)}

	if p.AfterArchive == "delete" {
		actions = append(actions, deleteAction(p, group, f, age))
	}

	return actions
}

func skipAction(p config.Policy, group string, f fsmodel.FileInfo, code, message string, facts map[string]string) plan.Action {
	return plan.Action{
		Kind:   plan.KindSkip,
		Policy: p.Name,
		Group:  group,
		Source: f.Path,
		Size:   f.Size,
		Reason: plan.Reason{
			Code:    code,
			Message: message,
			Facts:   facts,
		},
	}
}

func archiveAction(now time.Time, p config.Policy, group string, f fsmodel.FileInfo, age time.Duration) plan.Action {
	target := RenderArchiveTarget(p, group, now)
	return plan.Action{
		Kind:    plan.KindArchive,
		Policy:  p.Name,
		Group:   group,
		Source:  f.Path,
		Target:  target,
		Size:    f.Size,
		ModTime: f.ModTime,
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

func deleteAction(p config.Policy, group string, f fsmodel.FileInfo, age time.Duration) plan.Action {
	return plan.Action{
		Kind:    plan.KindDelete,
		Policy:  p.Name,
		Group:   group,
		Source:  f.Path,
		Size:    f.Size,
		ModTime: f.ModTime,
		Reason: plan.Reason{
			Code:    plan.ReasonAfterArchiveDelete,
			Message: fmt.Sprintf("delete source after successful archive (age=%s)", age),
			Facts: map[string]string{
				"age": age.String(),
			},
		},
	}
}

func archiveTarget(p config.Policy, group string, _ fsmodel.FileInfo) string {
	_ = group
	dest := p.Archive.Dest
	if dest == "" {
		dest = "/tmp/archive"
	}
	name := p.Archive.Name
	if name == "" {
		name = "{group}-{date}.zip"
	}
	return RenderArchiveTarget(p, group, time.Now().UTC())
}
