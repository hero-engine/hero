package drive

import "testing"

func TestRunLedgerRoundTripAndAnswer(t *testing.T) {
	dir := t.TempDir()
	l, err := LoadLedger(dir, "drive")
	if err != nil {
		t.Fatal(err)
	}
	if l.Pause != nil {
		t.Fatal("fresh ledger should have no pause")
	}
	l.SetPause(&PendingPause{Spec: "a", Category: "Supervised", Reason: "r"})
	if err := l.Save(); err != nil {
		t.Fatal(err)
	}

	l2, err := LoadLedger(dir, "drive")
	if err != nil {
		t.Fatal(err)
	}
	if l2.Pause == nil || l2.Pause.Spec != "a" {
		t.Fatalf("pause not persisted: %+v", l2.Pause)
	}
	paused, ok := l2.RecordAnswer("yes go")
	if !ok || paused != "a" {
		t.Fatalf("RecordAnswer = %q,%v want a,true", paused, ok)
	}
	if l2.Pause != nil {
		t.Error("pause should clear when answered")
	}
	if !l2.IsAnswered("a") {
		t.Error("a should be answered")
	}
	if err := l2.Save(); err != nil {
		t.Fatal(err)
	}

	l3, _ := LoadLedger(dir, "drive")
	if !l3.IsAnswered("a") {
		t.Error("answer should persist across reload")
	}
}

func TestRecordAnswerNoPause(t *testing.T) {
	l := &RunLedger{Initiative: "drive", Answered: map[string]string{}}
	if _, ok := l.RecordAnswer("x"); ok {
		t.Error("RecordAnswer with no open pause should report false")
	}
}
