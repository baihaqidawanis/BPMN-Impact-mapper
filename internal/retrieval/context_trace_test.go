package retrieval

import (
	"reflect"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/textutil"
)

func TestNeighbourTraceExplainsCandidateWithoutChangingScoreOrAction(t *testing.T) {
	artifact := domain.Artifact{ID: "handler", File: "requests.go", Symbol: domain.Symbol{Name: "SubmitRequest", Kind: "function", StartLine: 20},
		Content: "// STEP-10 is unrelated\n// STEP-1\nfunc SubmitRequest() {}", Tokens: textutil.Tokenize("submit request")}
	change := domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "STEP-NEW", Name: "Submit request"}}
	query := []string{"submit", "request"}
	before, ok := scoreArtifact(change, artifact, query, []string{"STEP-NEW"}, 1, 1, DefaultConfig())
	if !ok {
		t.Fatal("expected lexical candidate")
	}
	change.Context = &domain.ChangeContext{Predecessors: []domain.ElementContext{{ID: "STEP-1"}}, Successors: []domain.ElementContext{{ID: "STEP-1"}}}
	after, ok := scoreArtifact(change, artifact, query, []string{"STEP-NEW"}, 1, 1, DefaultConfig())
	if !ok || !reflect.DeepEqual(before.ScoreBreakdown, after.ScoreBreakdown) || after.RequiredAction != domain.ActionVerify {
		t.Fatalf("context annotation changed score/action: %#v", after)
	}
	count := 0
	for _, evidence := range after.Evidence {
		if evidence.Type == "TRACE_ID" {
			t.Fatal("neighbour was falsely classified as direct trace")
		}
		if evidence.Type == "CONTEXT_TRACE_ID" {
			count++
			if evidence.Location != "requests.go:21" || evidence.Excerpt != "// STEP-1" {
				t.Fatalf("incorrect neighbour evidence: %#v", evidence)
			}
		}
	}
	if count != 1 {
		t.Fatalf("expected one deduplicated context trace, got %d", count)
	}
}

func TestContextTraceRejectsCollisionsAndDoesNotDuplicateDirectTrace(t *testing.T) {
	change := domain.Change{Context: &domain.ChangeContext{Predecessors: []domain.ElementContext{{ID: "STEP-1"}}}}
	for _, content := range []string{"STEP-10", "XSTEP-1", "STEP-1_extra"} {
		if got := contextTraceEvidence(change, domain.Artifact{Content: content}, nil); len(got) != 0 {
			t.Fatalf("false neighbour match for %q", content)
		}
	}
	if got := contextTraceEvidence(change, domain.Artifact{Content: "STEP-1"}, []string{"step-1"}); len(got) != 0 {
		t.Fatal("direct trace duplicated as context")
	}
}
