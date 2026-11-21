package agentjobs

import (
	"context"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

type timedCheckpoints struct {
	planning.AgentSessionStore
	id             string
	start          time.Time
	previousMillis *int64
}

func (c timedCheckpoints) elapsed() *int64 {
	if c.previousMillis == nil {
		return nil
	}
	elapsed := *c.previousMillis + time.Since(c.start).Milliseconds()
	return &elapsed
}

func (c timedCheckpoints) Save(ctx context.Context, session planning.AgentSession) error {
	if session.ID == c.id {
		session.ActiveMillis = c.elapsed()
	}
	return c.AgentSessionStore.Save(ctx, session)
}

func (v View) withLiveElapsed() View {
	if !v.runStarted.IsZero() && v.ActiveMillis != nil {
		elapsed := *v.ActiveMillis + time.Since(v.runStarted).Milliseconds()
		v.ActiveMillis = &elapsed
		v.runStarted = time.Time{}
	}
	return v
}
