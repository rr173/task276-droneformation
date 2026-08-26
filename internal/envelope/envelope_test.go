package envelope

import (
	"math"
	"testing"

	"task276-droneformation/internal/model"
)

func TestCenterAtLinearExtrapolation(t *testing.T) {
	it := model.IntentSegment{TStart: 1000, PosX: 2, VelX: 3, PosY: 1, VelY: 0, PosZ: 0, VelZ: 0}
	c := CenterAt(it, 3000)
	if math.Abs(c.X-8) > 1e-9 || math.Abs(c.Y-1) > 1e-9 {
		t.Fatalf("center = %+v, want x=8 y=1", c)
	}
}

func TestRadiusAtUsesMaxSigmaTimesK(t *testing.T) {
	it := model.IntentSegment{TStart: 0, SigX: 0.4, SigY: 1.2, SigZ: 0.1}
	if got := RadiusAt(it, 0, 3); math.Abs(got-3.6) > 1e-9 {
		t.Fatalf("radius = %v, want 3.6", got)
	}
}
