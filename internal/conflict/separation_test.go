package conflict

import (
	"testing"

	"task276-droneformation/internal/model"
)

func TestEvaluatePairSafe(t *testing.T) {
	a := model.IntentSegment{AircraftID: "A", TStart: 0, TEnd: 10000, PosX: 0, SigX: 0.5, SigY: 0.5, SigZ: 0.5}
	b := model.IntentSegment{AircraftID: "B", TStart: 0, TEnd: 10000, PosX: 100, SigX: 0.5, SigY: 0.5, SigZ: 0.5}
	r := EvaluatePair(a, b, 0, 10000, 100, 3.0, 3.0)
	if r.Status != model.RelationSafe {
		t.Fatalf("expected safe, got %s", r.Status)
	}
}

func TestEvaluatePairInsufficient(t *testing.T) {
	a := model.IntentSegment{AircraftID: "A", TStart: 0, TEnd: 10000, PosX: 0, VelX: 1, SigX: 0.5}
	b := model.IntentSegment{AircraftID: "B", TStart: 0, TEnd: 10000, PosX: 5, SigX: 0.5}
	r := EvaluatePair(a, b, 0, 10000, 100, 3.0, 3.0)
	if r.Status != model.RelationInsufficient {
		t.Fatalf("expected insufficient, got %s (minEff=%.3f)", r.Status, r.MinEffDistance)
	}
}
