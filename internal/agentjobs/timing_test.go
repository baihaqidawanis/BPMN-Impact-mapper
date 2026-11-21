package agentjobs

import (
	"context"
	"testing"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

func TestCheckpointClockContinuesFromSavedActiveTime(t *testing.T) {
	ctx := context.Background()
	store := traceStore{}
	previousMillis := int64(3000)
	clock := timedCheckpoints{AgentSessionStore: store, id: "parent", start: time.Now().Add(-2 * time.Second), previousMillis: &previousMillis}
	for _, id := range []string{"parent", "child"} {
		if err := clock.Save(ctx, planning.AgentSession{ID: id, Status: "PAUSED"}); err != nil {
			t.Fatal(err)
		}
	}
	parent, err := store.Load(ctx, "parent")
	if err != nil || parent.ActiveMillis == nil || *parent.ActiveMillis < 5000 {
		t.Fatalf("missing accumulated active time: %v, %v", parent.ActiveMillis, err)
	}
	if store["child"].ActiveMillis != nil {
		t.Fatal("a child checkpoint recorded the parent clock")
	}
	view := viewSession(parent)
	if view.ActiveMillis == nil || *view.ActiveMillis != *parent.ActiveMillis {
		t.Fatal("paused view lost its checkpoint time")
	}
}
