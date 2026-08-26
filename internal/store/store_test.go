package store

import (
	"path/filepath"
	"testing"

	"task276-droneformation/internal/model"
)

func TestRunAndAircraftRoundTrip(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "roundtrip.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	run := &model.FormationRun{
		ID: "run-1", Name: "n", Status: model.RunReceiving,
		MinSeparationM: 2, ConfidenceK: 3, RuleVersion: 1, CreatedAt: 10,
	}
	if err := st.CreateRun(run); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetRun("run-1")
	if err != nil || got.Name != "n" {
		t.Fatalf("get run: %v %+v", err, got)
	}
	ac := &model.Aircraft{
		ID: "ac-1", RunID: "run-1", Callsign: "ALPHA",
		RadiusM: 0.5, Status: model.AircraftActive, CreatedAt: 10,
	}
	if err := st.CreateAircraft(ac); err != nil {
		t.Fatal(err)
	}
	gotAC, err := st.GetAircraft("ac-1")
	if err != nil || gotAC.Callsign != "ALPHA" {
		t.Fatalf("get aircraft: %v %+v", err, gotAC)
	}
}

func TestIntentSeqExists(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "seq.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	it := &model.IntentSegment{
		RunID: "r", AircraftID: "a", Seq: 7, TStart: 1, TEnd: 2,
		Status: model.IntentRaw, CreatedAt: 1,
	}
	if err := st.InsertIntent(it); err != nil {
		t.Fatal(err)
	}
	ok, err := st.IntentSeqExists("r", "a", 7)
	if err != nil || !ok {
		t.Fatalf("expected seq exists, err=%v ok=%v", err, ok)
	}
	ok, err = st.IntentSeqExists("r", "a", 8)
	if err != nil || ok {
		t.Fatalf("expected missing seq, err=%v ok=%v", err, ok)
	}
}
