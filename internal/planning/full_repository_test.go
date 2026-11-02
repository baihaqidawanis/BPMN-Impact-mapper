package planning

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type fullSourceFixture struct {
	fixtureTools
	chunks []RepositoryChunk
}

func (s fullSourceFixture) RepositoryChunks() []RepositoryChunk { return s.chunks }

func fullFixture() fullSourceFixture {
	s := fullSourceFixture{fixtureTools: fixtureTools{"full-fixture"}}
	for i := 0; i < 8; i++ {
		file := fmt.Sprintf("file%d.go", i)
		if i == 0 {
			file = "store.go"
		}
		content := "impact behavior"
		if i == 7 {
			content = "unrelated"
		}
		s.chunks = append(s.chunks, RepositoryChunk{ID: fmt.Sprintf("chunk-%d", i), File: file, Module: "core", Layer: "application", Content: content, EndByte: len(content), FileBytes: len(content), LastInFile: true})
	}
	return s
}

func TestFullRepositoryScanResumeAndAllFindingsPlanned(t *testing.T) {
	source := fullFixture()
	store := memorySessions{}
	calls := map[string]int{}
	fail := true
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if c := turn.RepositoryChunk; c != nil {
			calls[c.ID]++
			if c.ID == "chunk-1" && fail {
				fail = false
				return AgentReply{}, errors.New("connection lost")
			}
			reply, _ := goodReply(turn)
			if !strings.Contains(c.Content, "impact") {
				reply.Verification.Decisions[0].Verdict = VerdictReject
			}
			return reply, nil
		}
		if turn.Role == Analyst {
			t.Fatal("planning unexpectedly restarted analyst without a review request")
		}
		return goodReply(turn)
	}}
	engine := FullRepositoryEngine{Model: model, Source: source, Store: store}
	first, err := engine.Run(context.Background(), "full-run", agentBundle())
	if err == nil || first.Status != "PAUSED" || first.FullReview.ChunksReviewed != 1 {
		t.Fatal("interruption not checkpointed", first.Status, err)
	}
	changed := engine
	other := source
	other.fixtureTools.id = "different-source"
	changed.Source = other
	if _, err := changed.Run(context.Background(), "full-run", agentBundle()); err == nil {
		t.Fatal("source drift accepted")
	}
	final, err := engine.Run(context.Background(), "full-run", agentBundle())
	if err != nil || final.Status != "COMPLETE" {
		t.Fatal(final.Status, err, final.Feedback)
	}
	if calls["chunk-0"] != 1 || calls["chunk-1"] != 2 {
		t.Fatal("completed chunk resent", calls)
	}
	if final.FullReview.ChunksReviewed != 8 || final.FullReview.FilesReviewed != 8 || final.FullReview.BatchesReviewed != 3 || len(final.Draft) != 7 {
		t.Fatal("coverage/findings truncated", final.FullReview, len(final.Draft))
	}
	plan, err := final.ChangePlan()
	if err != nil || plan.FullReview == nil || !strings.Contains(string(Markdown(plan)), "Source chunks reviewed: 8/8") {
		t.Fatal("coverage export missing", err)
	}
	before := calls["chunk-0"]
	if _, err := engine.Run(context.Background(), "full-run", agentBundle()); err != nil || calls["chunk-0"] != before {
		t.Fatal("final replay performed inference", err)
	}
}

func TestFullRepositoryInvalidChunkNeverCountsAsCoverage(t *testing.T) {
	store := memorySessions{}
	source := fullFixture()
	source.chunks = source.chunks[:1]
	calls := 0
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		calls++
		reply, _ := goodReply(turn)
		reply.Verification.Decisions[0].EvidenceIDs = []string{"invented"}
		return reply, nil
	}}
	engine := FullRepositoryEngine{Model: model, Source: source, Store: store}
	var result AgentSession
	for i := 0; i < 4; i++ {
		result, _ = engine.Run(context.Background(), "invalid-full", agentBundle())
	}
	if calls != 3 || result.Status != "NEEDS_REVIEW" || result.FullReview.ChunksReviewed != 0 {
		t.Fatal("invalid output counted or retried indefinitely", calls, result.Status, result.FullReview)
	}
}

func TestFullRepositoryRejectsManifestGapsBeforeInference(t *testing.T) {
	source := fullSourceFixture{fixtureTools: fixtureTools{"gap"}, chunks: []RepositoryChunk{
		{ID: "a", File: "store.go", StartByte: 0, EndByte: 2, FileBytes: 4, Content: "ab"},
		{ID: "b", File: "store.go", StartByte: 3, EndByte: 4, FileBytes: 4, Content: "d", LastInFile: true},
	}}
	calls := 0
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) { calls++; return goodReply(turn) }}
	_, err := (FullRepositoryEngine{Model: model, Source: source, Store: memorySessions{}}).Run(context.Background(), "gap", agentBundle())
	if err == nil || calls != 0 {
		t.Fatal("incomplete byte coverage accepted", err, calls)
	}
}
