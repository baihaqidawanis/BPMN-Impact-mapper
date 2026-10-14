package retrieval_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/retrieval"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/textutil"
)

func TestAttachRequiresEvidenceAndExplainsStrongAction(t *testing.T) {
	content := `// ValidateRequest handles TASK-VALIDATE and writes an audit action.
func ValidateRequest() string { return "REVIEWED" }`
	artifact := domain.Artifact{
		ID: "internal/application/request.go#ValidateRequest", File: "internal/application/request.go",
		Module: "internal", Layer: "application", Language: "Go",
		Symbol:  domain.Symbol{Name: "ValidateRequest", Kind: "function", StartLine: 10},
		Content: content, Tokens: textutil.Tokenize("internal application request ValidateRequest " + content),
	}
	change := domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "TASK-VALIDATE", Name: "Validate request", Type: "serviceTask"}, Impacts: []domain.Impact{}, UnresolvedImpact: []string{}}
	changes, _ := retrieval.Attach([]domain.Change{change}, []domain.Artifact{artifact}, retrieval.DefaultConfig())
	if len(changes[0].Impacts) != 1 {
		t.Fatalf("impacts=%#v", changes[0].Impacts)
	}
	impact := changes[0].Impacts[0]
	if len(impact.Evidence) == 0 || impact.ScoreBreakdown.TraceID != 1 {
		t.Fatalf("impact lacks evidence: %#v", impact)
	}
	if impact.RequiredAction != domain.ActionAdd {
		t.Fatalf("action=%s confidence=%.4f breakdown=%#v", impact.RequiredAction, impact.Confidence, impact.ScoreBreakdown)
	}
}

func TestAttachAbstainsWithoutEvidence(t *testing.T) {
	change := domain.Change{ChangeType: domain.ModifyTaskLabel, NewElement: &domain.Element{ID: "TASK-X", Name: "Rare phrase", Type: "task"}, Impacts: []domain.Impact{}, UnresolvedImpact: []string{}}
	changes, _ := retrieval.Attach([]domain.Change{change}, nil, retrieval.DefaultConfig())
	if len(changes[0].Impacts) != 0 || len(changes[0].UnresolvedImpact) == 0 {
		t.Fatalf("expected abstention: %#v", changes[0])
	}
}

func TestLabelOnlyChangeWithExactTraceRequiresVerification(t *testing.T) {
	content := `func CreateRequest() string { return "PUR-PR-001" }`
	artifact := domain.Artifact{
		ID: "store.go#function:CreateRequest:1", File: "store.go", Module: ".", Layer: "application", Language: "Go",
		Symbol: domain.Symbol{Name: "CreateRequest", Kind: "function", StartLine: 1}, Content: content,
		Tokens: textutil.Tokenize("CreateRequest create classify purchase request " + content),
	}
	change := domain.Change{ChangeType: domain.ModifyTaskLabel, OldElement: &domain.Element{ID: "PUR-PR-001", Name: "Create purchase request", Type: "task"}, NewElement: &domain.Element{ID: "PUR-PR-001", Name: "Create and classify purchase request", Type: "task"}}
	changes, _ := retrieval.Attach([]domain.Change{change}, []domain.Artifact{artifact}, retrieval.DefaultConfig())
	if len(changes[0].Impacts) != 1 || changes[0].Impacts[0].Confidence < 0.70 {
		t.Fatalf("expected strong trace candidate: %#v", changes[0].Impacts)
	}
	if changes[0].Impacts[0].RequiredAction != domain.ActionVerify {
		t.Fatalf("label-only action=%s", changes[0].Impacts[0].RequiredAction)
	}
}

func TestAttachDropsPackageCandidateWhenSpecificSymbolMatches(t *testing.T) {
	content := `func CreateRequest() string { return "PUR-PR-001" }`
	artifacts := []domain.Artifact{
		{ID: "store.go", File: "store.go", Module: ".", Layer: "application", Language: "Go", Symbol: domain.Symbol{Name: "purchasing", Kind: "package"}, Content: content, Tokens: textutil.Tokenize(content)},
		{ID: "store.go#function:CreateRequest:1", File: "store.go", Module: ".", Layer: "application", Language: "Go", Symbol: domain.Symbol{Name: "CreateRequest", Kind: "function", StartLine: 1}, Content: content, Tokens: textutil.Tokenize("CreateRequest " + content)},
	}
	change := domain.Change{ChangeType: domain.ModifyTaskLabel, NewElement: &domain.Element{ID: "PUR-PR-001", Name: "Create purchase request", Type: "task"}}
	changes, _ := retrieval.Attach([]domain.Change{change}, artifacts, retrieval.DefaultConfig())
	if len(changes[0].Impacts) != 1 || changes[0].Impacts[0].Symbol != "CreateRequest" {
		t.Fatalf("impacts=%#v", changes[0].Impacts)
	}
	bm25Changes, _, err := (retrieval.BM25Only{}).Attach(context.Background(), []domain.Change{change}, artifacts, retrieval.Config{TopK: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(bm25Changes[0].Impacts) != 1 || bm25Changes[0].Impacts[0].Symbol != "CreateRequest" {
		t.Fatalf("BM25 impacts=%#v", bm25Changes[0].Impacts)
	}
}

func TestBM25OnlyProducesVerifyCandidatesDeterministically(t *testing.T) {
	artifacts := []domain.Artifact{
		{ID: "b", File: "b.go", Module: ".", Layer: "unknown", Content: "validate request", Tokens: []string{"validate", "request"}},
		{ID: "a", File: "a.go", Module: ".", Layer: "unknown", Content: "validate request", Tokens: []string{"validate", "request"}},
	}
	change := domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "TASK", Name: "Validate request", Type: "task"}, Impacts: []domain.Impact{}, UnresolvedImpact: []string{}}
	first, warnings, err := (retrieval.BM25Only{}).Attach(context.Background(), []domain.Change{change}, artifacts, retrieval.Config{TopK: 2})
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := (retrieval.BM25Only{}).Attach(context.Background(), []domain.Change{change}, artifacts, retrieval.Config{TopK: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 || len(first[0].Impacts) != 2 {
		t.Fatalf("impacts=%#v warnings=%#v", first[0].Impacts, warnings)
	}
	if first[0].Impacts[0].File != "a.go" || first[0].Impacts[0].RequiredAction != domain.ActionVerify || first[0].Impacts[0].Evidence[0].Type != "BM25" {
		t.Fatalf("first candidate=%#v", first[0].Impacts[0])
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("BM25-only output is not deterministic")
	}
}

func TestCompositeContextContributesNeighborTermsToRetrieval(t *testing.T) {
	content := `func RegisterInvoice() { PrepareInvoice() }`
	artifact := domain.Artifact{
		ID: "invoice.go#function:RegisterInvoice:1", File: "internal/application/invoice.go", Module: "internal", Layer: "application",
		Symbol: domain.Symbol{Name: "RegisterInvoice", Kind: "function", StartLine: 1}, Content: content,
		Tokens: textutil.Tokenize("internal application invoice RegisterInvoice PrepareInvoice " + content),
	}
	change := domain.Change{
		ChangeType: domain.AddTask,
		NewElement: &domain.Element{ID: "TASK-X", Name: "Manual checkpoint", Type: "userTask"},
		Context: &domain.ChangeContext{
			Pattern:      domain.PatternInsertTask,
			Predecessors: []domain.ElementContext{{ID: "TASK-PREPARE", Name: "Prepare invoice", Type: "userTask"}},
			Successors:   []domain.ElementContext{{ID: "TASK-REGISTER", Name: "Register invoice", Type: "serviceTask"}},
		},
	}
	changes, _ := retrieval.Attach([]domain.Change{change}, []domain.Artifact{artifact}, retrieval.DefaultConfig())
	if len(changes[0].Impacts) != 1 || changes[0].Impacts[0].File != artifact.File {
		t.Fatalf("contextual impacts=%#v", changes[0].Impacts)
	}
	if changes[0].Impacts[0].ScoreBreakdown.Symbol == 0 || changes[0].Impacts[0].ScoreBreakdown.BM25 == 0 {
		t.Fatalf("context did not affect score=%#v", changes[0].Impacts[0].ScoreBreakdown)
	}
}

func TestTraceIDUsesIdentifierBoundaries(t *testing.T) {
	change := domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "PUR-PR-001", Name: "Create purchase request", Type: "userTask"}}
	artifacts := []domain.Artifact{
		{ID: "exact", File: "exact.yaml", Content: "operationId: PUR-PR-0010-collision\noperationId: PUR-PR-001-create", Tokens: textutil.Tokenize("create purchase request")},
		{ID: "numeric-prefix", File: "numeric.go", Content: `const trace = "PUR-PR-0010"`, Tokens: textutil.Tokenize("unrelated")},
		{ID: "leading-prefix", File: "leading.go", Content: `const trace = "XPUR-PR-001"`, Tokens: textutil.Tokenize("unrelated")},
		{ID: "underscore-prefix", File: "underscore.go", Content: `const trace = "PUR-PR-001_extra"`, Tokens: textutil.Tokenize("unrelated")},
	}
	changes, _ := retrieval.Attach([]domain.Change{change}, artifacts, retrieval.DefaultConfig())
	if len(changes[0].Impacts) != 1 || changes[0].Impacts[0].File != "exact.yaml" {
		t.Fatalf("trace boundary impacts=%#v", changes[0].Impacts)
	}
	if changes[0].Impacts[0].ScoreBreakdown.TraceID != 1 {
		t.Fatalf("exact operation trace not recognized: %#v", changes[0].Impacts[0])
	}
	for _, evidence := range changes[0].Impacts[0].Evidence {
		if evidence.Type == "TRACE_ID" && evidence.Location != "exact.yaml:2" {
			t.Fatalf("trace evidence points to prefix collision: %#v", evidence)
		}
	}
}

func TestPathAndSymbolRecognizeConservativeInflections(t *testing.T) {
	artifact := domain.Artifact{
		ID: "purchasing", File: "internal/purchasing/requests.go", Module: "internal", Layer: "application",
		Symbol: domain.Symbol{Name: "Requests", Kind: "function"}, Content: "unrelated", Tokens: []string{"unrelated"},
	}
	change := domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "TASK-X", Name: "Purchase request", Type: "userTask"}}
	config := retrieval.DefaultConfig()
	config.CandidateThreshold = 0.01
	changes, _ := retrieval.Attach([]domain.Change{change}, []domain.Artifact{artifact}, config)
	if len(changes[0].Impacts) != 1 {
		t.Fatalf("inflected candidate missing: %#v", changes[0])
	}
	score := changes[0].Impacts[0].ScoreBreakdown
	if score.Path == 0 || score.Symbol == 0 || score.Lexical != 0 || score.TraceID != 0 {
		t.Fatalf("unexpected inflection score=%#v", score)
	}
}

func TestInflectionMatchingRejectsSharedPrefixes(t *testing.T) {
	artifact := domain.Artifact{
		ID: "prefix", File: "internal/statement/username.go", Module: "internal", Layer: "application",
		Symbol: domain.Symbol{Name: "Username", Kind: "function"}, Content: "unrelated", Tokens: []string{"unrelated"},
	}
	change := domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "TASK-X", Name: "State user", Type: "userTask"}}
	config := retrieval.DefaultConfig()
	config.CandidateThreshold = 0.01
	changes, _ := retrieval.Attach([]domain.Change{change}, []domain.Artifact{artifact}, config)
	if len(changes[0].Impacts) != 0 {
		t.Fatalf("shared prefixes must not become evidence: %#v", changes[0].Impacts)
	}
}
