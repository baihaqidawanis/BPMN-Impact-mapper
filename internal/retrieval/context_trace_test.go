package retrieval

import (
	"reflect"
	"slices"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/textutil"
)

func TestQueryKeepsBusinessWordsButNotGeneratedGraphIdentifiers(t *testing.T) {
	change := domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{
		ID: "FLOW-PR-003A", Name: "Clarify purchase request", Actor: "Operator",
		Incoming: []string{"FLOW-PR-002"}, Outgoing: []string{"FLOW-PR-003B"},
	}, Context: &domain.ChangeContext{Predecessors: []domain.ElementContext{{ID: "FLOW-PR-001", Name: "Submit purchase request"}}}}
	query, ids := queryFor(change)
	if !slices.Contains(query, "purchase") || !slices.Contains(query, "clarify") || !slices.Contains(query, "submit") {
		t.Fatalf("business/context terms missing: %v", query)
	}
	for _, noise := range []string{"flow", "003a", "operator", "add", "task"} {
		if slices.Contains(query, noise) {
			t.Fatalf("structural metadata leaked into lexical query: %s in %v", noise, query)
		}
	}
	if !reflect.DeepEqual(ids, []string{"FLOW-PR-003A"}) {
		t.Fatalf("direct trace ID was lost: %v", ids)
	}
}

func TestStructuralQueryDoesNotBorrowNeighbourWords(t *testing.T) {
	change := domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{Name: "Upload supporting document for request"},
		Context: &domain.ChangeContext{Predecessors: []domain.ElementContext{{Name: "Select product"}}}}
	query, _ := queryFor(change)
	if !slices.Contains(query, "product") {
		t.Fatal("neighbour context was lost from retrieval query")
	}
	structural := structuralQueryFor(change, query)
	if slices.Contains(structural, "product") || !slices.Contains(structural, "document") {
		t.Fatalf("structural query borrowed neighbour terms: %v", structural)
	}
	artifact := domain.Artifact{File: "store.go", Content: `UPDATE products SET note='request'`, Tokens: textutil.Tokenize(`UPDATE products SET note='request'`)}
	impact, ok := scoreArtifact(change, artifact, query, nil, 1, 1, Config{CandidateThreshold: 0.01})
	if !ok || impact.ScoreBreakdown.Structured != 0 {
		t.Fatalf("neighbour-only product earned structural evidence: %+v", impact)
	}
	for _, evidence := range impact.Evidence {
		if evidence.Type == "DATABASE_TABLE" {
			t.Fatalf("misleading table evidence: %+v", evidence)
		}
	}
}

func TestLexicalEvidenceUsesBestMatchingLineOrPath(t *testing.T) {
	artifact := domain.Artifact{File: "ui/purchase-request.tsx", Content: "request is generic\nvalidate purchase request details\nrequest again"}
	query := []string{"validate", "purchase", "request", "details"}
	got := evidenceAtQuery("BM25", artifact, query)
	if got.Location != "ui/purchase-request.tsx:2" || got.Excerpt != "validate purchase request details" {
		t.Fatalf("wrong lexical support: %#v", got)
	}
	got = evidenceAtQuery("BM25", domain.Artifact{File: "ui/purchase-request.tsx", Content: "unrelated text"}, query)
	if got.Location != "ui/purchase-request.tsx" || got.Excerpt != "ui/purchase-request.tsx" {
		t.Fatalf("path-only match cited unrelated source: %#v", got)
	}
}

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
