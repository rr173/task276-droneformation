package intent

import (
	"testing"

	"task276-droneformation/internal/model"
)

func TestCommonWindowIntersection(t *testing.T) {
	intents := []model.IntentSegment{
		{TStart: 100, TEnd: 500},
		{TStart: 200, TEnd: 800},
	}
	start, end, ok := CommonWindow(intents)
	if !ok || start != 200 || end != 500 {
		t.Fatalf("window = %d,%d,%v; want 200,500,true", start, end, ok)
	}
}

func TestLatestActiveUsesHighestSeq(t *testing.T) {
	now := int64(1000)
	segs := []model.IntentSegment{
		{Seq: 1, TEnd: 2000, CreatedAt: now},
		{Seq: 3, TEnd: 2000, CreatedAt: now},
		{Seq: 2, TEnd: 2000, CreatedAt: now},
	}
	got, ok := LatestActive(segs, now)
	if !ok || got.Seq != 3 {
		t.Fatalf("latest seq=%d ok=%v, want 3/true", got.Seq, ok)
	}
}

func TestSampleStepBounded(t *testing.T) {
	step := SampleStep(0, 100)
	if step < 50 {
		t.Fatalf("step %d below minimum", step)
	}
}
