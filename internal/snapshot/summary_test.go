package snapshot

import (
	"testing"

	"task276-droneformation/internal/model"
)

func TestBuildSummaryConflictWins(t *testing.T) {
	sum := BuildSummary([]model.AvoidanceRelation{
		{Status: model.RelationSafe},
		{Status: model.RelationInsufficient},
		{Status: model.RelationSafe},
	})
	if sum.ConflictCount != 1 || sum.SafeCount != 2 || sum.Status != model.RunConflict {
		t.Fatalf("summary = %+v", sum)
	}
}

func TestBuildSummaryAllSafe(t *testing.T) {
	sum := BuildSummary([]model.AvoidanceRelation{
		{Status: model.RelationSafe},
		{Status: model.RelationSafe},
	})
	if sum.ConflictCount != 0 || sum.Status != model.RunSafe {
		t.Fatalf("summary = %+v", sum)
	}
}
