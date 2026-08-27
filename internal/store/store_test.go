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

// TestListIntentsByRunRoundTrip 钉死 ListIntentsByRun 的两个不变量：
//  1. 必须真正读出数据（曾因漏迭代而恒返回空切片）；
//  2. 查询返回后游标必须关闭，不能占住 SQLite 单连接写锁（曾导致
//     “列出全编队意图后再触发验证一直不结束”，读游标未释放、写事务拿不到锁而挂起）。
func TestListIntentsByRunRoundTrip(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "listrun.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	for i, ac := range []string{"a", "b"} {
		it := &model.IntentSegment{
			RunID: "r", AircraftID: ac, Seq: int64(i + 1), TStart: 1, TEnd: 2,
			Status: model.IntentRaw, CreatedAt: int64(i + 1),
		}
		if err := st.InsertIntent(it); err != nil {
			t.Fatal(err)
		}
	}

	got, err := st.ListIntentsByRun("r")
	if err != nil {
		t.Fatalf("list intents by run: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 intents, got %d", len(got))
	}
	if got[0].AircraftID != "a" || got[1].AircraftID != "b" {
		t.Fatalf("unexpected order: %+v", got)
	}

	// 读游标已随返回释放：后续写入必须立即成功，否则说明游标泄漏占住了写锁。
	it := &model.IntentSegment{
		RunID: "r", AircraftID: "c", Seq: 1, TStart: 1, TEnd: 2,
		Status: model.IntentRaw, CreatedAt: 99,
	}
	if err := st.InsertIntent(it); err != nil {
		t.Fatalf("write after list failed (cursor leak): %v", err)
	}
}
