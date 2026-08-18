package policy

import (
	"fmt"
	"sort"

	"example/log-retention/internal/config"
	"example/log-retention/internal/fsmodel"
)

type FileMatch struct {
	Path     string
	Policy   config.Policy
	Priority int
}

func BuildFilePolicyMap(snap fsmodel.Snapshot, policies []config.Policy) (map[string][]config.Policy, error) {
	result := make(map[string][]config.Policy)

	for _, p := range policies {
		candidates, err := SelectCandidates(snap, p)
		if err != nil {
			return nil, fmt.Errorf("policy %q: %w", p.Name, err)
		}
		for _, f := range candidates {
			result[f.Path] = append(result[f.Path], p)
		}
	}

	return result, nil
}

func ResolveConflicts(filePolicyMap map[string][]config.Policy) (map[string]string, []Conflict) {
	winners := make(map[string]string)
	var conflicts []Conflict

	paths := make([]string, 0, len(filePolicyMap))
	for path := range filePolicyMap {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		policies := filePolicyMap[path]
		if len(policies) == 0 {
			continue
		}
		if len(policies) == 1 {
			winners[path] = policies[0].Name
			continue
		}

		winner, isConflict := selectWinner(policies)
		if isConflict {
			conflicts = append(conflicts, Conflict{
				Path:     path,
				Policies: policyNames(policies),
			})
		} else {
			winners[path] = winner
		}
	}

	return winners, conflicts
}

type Conflict struct {
	Path     string   `json:"path"`
	Policies []string `json:"policies"`
}

func (c Conflict) String() string {
	return fmt.Sprintf("ambiguous file %s: policies %v", c.Path, c.Policies)
}

func selectWinner(policies []config.Policy) (winner string, isConflict bool) {
	maxPriority := policies[0].Priority
	for _, p := range policies[1:] {
		if p.Priority > maxPriority {
			maxPriority = p.Priority
		}
	}

	var topPolicies []config.Policy
	for _, p := range policies {
		if p.Priority == maxPriority {
			topPolicies = append(topPolicies, p)
		}
	}

	if len(topPolicies) == 1 {
		return topPolicies[0].Name, false
	}

	return "", true
}

func policyNames(policies []config.Policy) []string {
	names := make([]string, 0, len(policies))
	for _, p := range policies {
		names = append(names, p.Name)
	}
	sort.Strings(names)
	return names
}
