package planning

import (
	"context"
	"testing"
)

type readmeFixture struct{ fullSourceFixture }

func (readmeFixture) RootReadme() *ReadmeInput {
	return &ReadmeInput{File: "README.md", Content: "Application overview"}
}
func TestReadmeOrientationOnceAndNotImplementationEvidence(t *testing.T) {
	calls := 0
	store := memorySessions{}
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if turn.Readme != nil {
			calls++
			for _, c := range turn.Bundle.Changes {
				if len(c.Candidates) != 0 {
					t.Fatal("orientation received impact candidates")
				}
			}
			return AgentReply{Verification: &VerificationResponse{RepositorySummary: "README describes an adjustment application."}}, nil
		}
		return goodReply(turn)
	}}
	source := readmeFixture{fullFixture()}
	source.chunks = source.chunks[:1]
	engine := FullRepositoryEngine{Model: model, Source: source, Store: store}
	first, err := engine.Run(context.Background(), "readme-once", agentBundle())
	if err != nil || first.Status != "COMPLETE" {
		t.Fatal(err, first.Status)
	}
	if first.Bundle.RepositoryOverview == nil || first.Bundle.RepositoryOverview.File != "README.md" {
		t.Fatal("summary missing")
	}
	for _, c := range first.Bundle.Changes {
		for _, candidate := range c.Candidates {
			if candidate.File == "README.md" {
				t.Fatal("README promoted to implementation evidence")
			}
		}
	}
	if _, err := engine.Run(context.Background(), "readme-once", agentBundle()); err != nil || calls != 1 {
		t.Fatal("README reread", calls, err)
	}
}
