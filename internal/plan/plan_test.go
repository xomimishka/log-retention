package plan

import (
	"bytes"
	"testing"
	"time"
)

func TestMarshalPlanJSONDeterministic(t *testing.T) {
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)

	makePlan := func(firstSource, secondSource string) Plan {
		return Plan{
			PlanVersion: PlanVersion,
			Now:         now,
			Config:      "lrt.yaml",
			Actions: []Action{
				{
					Kind:   KindArchive,
					Policy: "service-logs",
					Group:  "service",
					Source: firstSource,
					Reason: Reason{Code: ReasonGenerationOverLimit, Message: "test"},
				},
				{
					Kind:   KindArchive,
					Policy: "service-logs",
					Group:  "service",
					Source: secondSource,
					Reason: Reason{Code: ReasonGenerationOverLimit, Message: "test"},
				},
			},
		}
	}

	p1 := makePlan("/var/log/a.log.3", "/var/log/a.log.4")
	p2 := makePlan("/var/log/a.log.4", "/var/log/a.log.3")

	b1, err := MarshalPlanJSON(p1)
	if err != nil {
		t.Fatal(err)
	}

	b2, err := MarshalPlanJSON(p2)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(b1, b2) {
		t.Fatalf("plan JSON is not deterministic\n\nfirst:\n%s\nsecond:\n%s", b1, b2)
	}
}

func TestPlanNormalizeArchiveBeforeDelete(t *testing.T) {
	p := Plan{
		PlanVersion: PlanVersion,
		Now:         time.Now().UTC(),
		Actions: []Action{
			{
				Kind:   KindDelete,
				Policy: "p1",
				Source: "/a.log",
				Reason: Reason{Code: ReasonAfterArchiveDelete},
			},
			{
				Kind:   KindArchive,
				Policy: "p1",
				Source: "/a.log",
				Reason: Reason{Code: ReasonGenerationOverLimit},
			},
		},
	}

	b, err := MarshalPlanJSON(p)
	if err != nil {
		t.Fatal(err)
	}

	if p.Actions[0].Kind != KindArchive {
		t.Fatalf("archive should come before delete, got JSON:\n%s", b)
	}
}
