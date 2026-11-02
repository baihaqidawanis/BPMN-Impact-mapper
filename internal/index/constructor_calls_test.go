package index_test

import (
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/index"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"strings"
	"testing"
)

func TestConstructorCallsUseExplicitProductionBindingAndAbstainWhenUnknown(t *testing.T) {
	for _, literal := range []string{"Service{s}", "Service{store:s}"} {
		for _, extra := range []string{"", "; _ = Build(Alternate{})", "; _ = Build(unknown)"} {
			code := `package sample
type Store interface { CopyInvoice(string) error }
type Concrete struct{}
func(Concrete)CopyInvoice(string)error{return nil}
type Alternate struct{}
func(Alternate)CopyInvoice(string)error{return nil}
type Service struct{store Store}
func Build(s Store)Service{return ` + literal + `}
func(s Service)CopyInvoice(id string)error{return s.store.CopyInvoice(id)}
func Compose(){_ = Build(Concrete{})` + extra + `}
`
			r := index.DefaultRegistry()
			r.Extractors = append(r.Extractors, index.GoWiringExtractor{})
			got := r.Build(ingest.Inventory{Files: []ingest.File{{RelativePath: "sample.go", Content: []byte(code)}}})
			links := []domain.SourceRelation{}
			for _, a := range got.Artifacts {
				for _, e := range a.Relations {
					if e.Kind == domain.RelationConstructorCall {
						links = append(links, e)
					}
				}
			}
			if extra != "" {
				if len(links) != 0 {
					t.Fatalf("unknown/ambiguous binding linked: %+v", links)
				}
			} else {
				if len(links) != 1 || links[0].BindingLocation == "" || !strings.Contains(links[0].BindingExcerpt, "Build(Concrete{})") || links[0].TargetLocation == "" {
					t.Fatalf("missing constructor provenance: %+v", links)
				}
			}
		}
	}
}

func TestConstructorCallsAbstainOnDirectConstructionOrFieldMutation(t *testing.T) {
	for _, extra := range []string{`func Other(){_ = Service{Concrete{}}}`, `func Replace(s *Service){s.store=Concrete{}}`} {
		code := `package sample
type Store interface { CopyInvoice(string) error }
type Concrete struct{}
func(Concrete)CopyInvoice(string)error{return nil}
type Service struct{store Store}
func Build(s Store)Service{return Service{s}}
func(s Service)CopyInvoice(id string)error{return s.store.CopyInvoice(id)}
func Compose(){_ = Build(Concrete{})}
` + extra
		r := index.SourceContextRegistry()
		got := r.Build(ingest.Inventory{Files: []ingest.File{{RelativePath: "source.go", Content: []byte(code)}}})
		for _, a := range got.Artifacts {
			for _, e := range a.Relations {
				if e.Kind == domain.RelationConstructorCall {
					t.Fatal("accepted unsupported alternative binding/mutation", extra)
				}
			}
		}
	}
}
