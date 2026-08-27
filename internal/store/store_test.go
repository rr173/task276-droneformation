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

// TestListIntentsByAircraftNoAliasing 回归：先列甲机意图，再列乙机意图，
// 两者返回的切片不得共享底层数组——否则甲机的列表会被乙机的数据覆盖。
func TestListIntentsByAircraftNoAliasing(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "aliasing.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	for _, ac := range []string{"jia", "yi"} {
		// 甲机与乙机各插两条意图，用 X 坐标区分所属飞行器。
		for seq := int64(1); seq <= 2; seq++ {
			posX := 1.0
			if ac == "yi" {
				posX = 9.0
			}
			if err := st.InsertIntent(&model.IntentSegment{
				RunID: "r", AircraftID: ac, Seq: seq, TStart: 1, TEnd: 2,
				PosX: posX, Status: model.IntentRaw, CreatedAt: 1,
			}); err != nil {
				t.Fatal(err)
			}
		}
	}

	jia, err := st.ListIntentsByAircraft("r", "jia")
	if err != nil {
		t.Fatal(err)
	}
	yi, err := st.ListIntentsByAircraft("r", "yi")
	if err != nil {
		t.Fatal(err)
	}
	// 先列出的甲机切片内容不得因随后列出乙机而被改写。
	if len(jia) != 2 || jia[0].AircraftID != "jia" || jia[0].PosX != 1.0 {
		t.Fatalf("jia slice corrupted after listing yi: %+v", jia)
	}
	if len(yi) != 2 || yi[0].AircraftID != "yi" || yi[0].PosX != 9.0 {
		t.Fatalf("yi slice unexpected: %+v", yi)
	}
}
