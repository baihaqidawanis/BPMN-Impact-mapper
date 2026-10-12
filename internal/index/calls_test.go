package index_test

import (
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/index"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/retrieval"
	"reflect"
	"strings"
	"testing"
)

func TestCallsResolveImportsAndRejectShadowing(t *testing.T) {
	inv := ingest.Inventory{Files: []ingest.File{
		{RelativePath: "go.mod", Content: []byte("module example.local/demo\n")},
		{RelativePath: "service/run.go", Content: []byte("package service\nfunc Run() {}\n")},
		{RelativePath: "api/handler.go", Content: []byte(`package api
import svc "example.local/demo/service"
func Handle() { svc.Run(); helper() }
func helper() {}
func Shadow(svc interface{ Run() }) { svc.Run() }
func Local(helper func()) { helper() }
`)},
	}}
	first, _ := index.Build(inv)
	second, _ := index.Build(inv)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("non deterministic calls")
	}
	found := false
	for _, a := range first {
		if a.Symbol.Name == "Handle" {
			found = true
			if len(a.Relations) != 2 {
				t.Fatalf("calls=%+v", a.Relations)
			}
			for _, c := range a.Relations {
				if c.Location != "api/handler.go:3" || c.Excerpt == "" {
					t.Fatalf("missing call-site evidence: %+v", c)
				}
			}
		}
		if (a.Symbol.Name == "Shadow" || a.Symbol.Name == "Local") && len(a.Relations) != 0 {
			t.Fatalf("shadowed call resolved: %+v", a)
		}
	}
	if !found {
		t.Fatal("missing handler")
	}
}

func TestIndexedCallRecoversCandidateWithoutQueryTerms(t *testing.T) {
	inv := ingest.Inventory{Files: []ingest.File{
		{RelativePath: "a.go", Content: []byte("package p\nfunc ValidateRequest(){ println(\"TASK-VALIDATE\"); Persist() }")},
		{RelativePath: "b.go", Content: []byte("package p\nfunc Persist(){}")},
	}}
	artifacts, _ := index.Build(inv)
	change := domain.Change{ChangeType: domain.ModifyTaskLabel, NewElement: &domain.Element{ID: "TASK-VALIDATE", Name: "Validate request", Type: "task"}}
	first, _ := retrieval.Attach([]domain.Change{change}, artifacts, retrieval.DefaultConfig())
	second, _ := retrieval.Attach([]domain.Change{change}, artifacts, retrieval.DefaultConfig())
	if !reflect.DeepEqual(first, second) {
		t.Fatal("propagated output changed on rerun")
	}
	for _, impact := range first[0].Impacts {
		if impact.Symbol == "Persist" {
			if impact.RequiredAction != domain.ActionVerify || impact.ScoreBreakdown.Dependency == 0 || impact.Evidence[0].Type != "CALLED_BY" {
				t.Fatalf("unexpected impact %+v", impact)
			}
			return
		}
	}
	t.Fatalf("missing dependency candidate: %+v", first)
}

func TestDuplicateTargetsAbstain(t *testing.T) {
	inv := ingest.Inventory{Files: []ingest.File{
		{RelativePath: "a.go", Content: []byte("package p\nfunc A(){ B() }\nfunc B(){}")},
		{RelativePath: "b.go", Content: []byte("package p\nfunc B(){}")},
	}}
	artifacts, _ := index.Build(inv)
	for _, a := range artifacts {
		if len(a.Relations) > 0 {
			t.Fatal("ambiguous target resolved")
		}
	}
}

func TestTypeAwareCallsResolveReceiverFieldAndUniqueInterface(t *testing.T) {
	inv := ingest.Inventory{Files: []ingest.File{
		{RelativePath: "go.mod", Content: []byte("module example.local/demo\n")},
		{RelativePath: "application/service.go", Content: []byte(`package application
type Service interface {
	Execute(DTO) error
	Ready() bool
}
type DTO struct { Value string }
`)},
		{RelativePath: "adapter/store.go", Content: []byte(`package adapter
import app "example.local/demo/application"
type Store struct{}
func (s *Store) Execute(value app.DTO) error { println("TASK-EXECUTE"); return nil }
func (s *Store) Ready() bool { return true }
func (s *Store) Direct(value app.DTO) error { return s.Execute(value) }
func Local(value app.DTO) error { store := &Store{}; return store.Execute(value) }
`)},
		{RelativePath: "transport/handler.go", Content: []byte(`package transport
import app "example.local/demo/application"
type Handler struct { service app.Service }
func (h Handler) Handle(value app.DTO) error { println("TASK-HANDLER"); return h.service.Execute(value) }
`)},
	}}
	first, _ := index.Build(inv)
	second, _ := index.Build(inv)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("type-aware relations are not deterministic")
	}
	assertRelation := func(symbol string, kind domain.RelationKind, target string) {
		t.Helper()
		for _, artifact := range first {
			if artifact.Symbol.Name != symbol {
				continue
			}
			for _, relation := range artifact.Relations {
				if relation.Kind == kind && strings.Contains(relation.TargetID, target) && relation.Location != "" && relation.Excerpt != "" {
					return
				}
			}
			t.Fatalf("missing %s relation from %s: %+v", kind, symbol, artifact.Relations)
		}
		t.Fatalf("missing symbol %s", symbol)
	}
	assertRelation("Direct", domain.RelationCallsMethod, "#method:Execute:")
	assertRelation("Local", domain.RelationCallsMethod, "#method:Execute:")
	assertRelation("Handle", domain.RelationInterfaceDispatch, "#method:Execute:")

	change := domain.Change{ChangeType: domain.ModifyActorRole, NewElement: &domain.Element{ID: "TASK-EXECUTE", Name: "Execute", Type: "task"}}
	changes, _ := retrieval.Attach([]domain.Change{change}, first, retrieval.DefaultConfig())
	if !hasImpactEvidence(changes[0].Impacts, "Handler.Handle", "INTERFACE_DISPATCH") {
		t.Fatalf("missing interface-propagated handler: %+v", changes[0].Impacts)
	}

	reverseChange := domain.Change{ChangeType: domain.ModifyActorRole, NewElement: &domain.Element{ID: "TASK-HANDLER", Name: "Orchestrate uncommon", Type: "task"}}
	reverse, _ := retrieval.Attach([]domain.Change{reverseChange}, first, retrieval.DefaultConfig())
	if !hasImpactEvidence(reverse[0].Impacts, "Store.Execute", "IMPLEMENTS_INTERFACE") {
		t.Fatalf("missing unique implementation candidate: %+v", reverse[0].Impacts)
	}
}

func hasImpactEvidence(impacts []domain.Impact, symbol, evidenceType string) bool {
	for _, impact := range impacts {
		if impact.Symbol == symbol && len(impact.Evidence) > 0 && impact.Evidence[0].Type == evidenceType && impact.RequiredAction == domain.ActionVerify {
			return true
		}
	}
	return false
}

func TestInterfaceDispatchAbstainsForAmbiguousOrEmbeddedInterface(t *testing.T) {
	inv := ingest.Inventory{Files: []ingest.File{
		{RelativePath: "ambiguous.go", Content: []byte(`package p
type Service interface { Execute(string) error }
type A struct{}
func (A) Execute(string) error { return nil }
type B struct{}
func (B) Execute(string) error { return nil }
func Ambiguous(service Service) error { return service.Execute("x") }

type Base interface { Execute(string) error }
type Extended interface { Base }
func Embedded(service Extended) error { return service.Execute("x") }

type NumberService interface { Execute(int) error }
type Wrong struct{}
func (Wrong) Execute(string) error { return nil }
func WrongSignature(service NumberService) error { return service.Execute(1) }

func ShadowReceiver(service A) error {
	{ service := B{}; return service.Execute("x") }
}
func Closure(service A) {
	func(service B) { _ = service.Execute("x") }(B{})
}
`)},
	}}
	artifacts, _ := index.Build(inv)
	for _, artifact := range artifacts {
		if artifact.Symbol.Name == "Ambiguous" || artifact.Symbol.Name == "Embedded" || artifact.Symbol.Name == "WrongSignature" || artifact.Symbol.Name == "ShadowReceiver" || artifact.Symbol.Name == "Closure" {
			if len(artifact.Relations) != 0 {
				t.Fatalf("unsupported interface dispatch was guessed: %+v", artifact)
			}
		}
	}
}
