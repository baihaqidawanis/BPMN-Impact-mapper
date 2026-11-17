package filestore

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/reportstore"
	"testing"
)

func TestBPMNInputsRestoreCompleteProcessWithoutLeakingXMLIntoReport(t *testing.T) {
	ctx := context.Background()
	store := ReportStore{Root: t.TempDir()}
	id := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	report := domain.Report{AnalysisID: id, InputFingerprint: "snapshot"}
	if err := store.Save(ctx, report); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetBPMN(ctx, id); !errors.Is(err, reportstore.ErrNotFound) {
		t.Fatal(err)
	}
	xml := `<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"><process id="p"><startEvent id="s"/><userTask id="t" name="Unchanged task"/><sequenceFlow id="f" sourceRef="s" targetRef="t"/></process></definitions>`
	pair := domain.BPMNPair{AnalysisID: id, InputFingerprint: "snapshot", BeforeXML: xml, AfterXML: xml}
	if err := store.SaveBPMN(ctx, pair); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.ProcessContext == nil || got.ProcessContext.Before.Elements["t"].Name != "Unchanged task" || got.ProcessContext.After.Flows["f"].TargetRef != "t" {
		t.Fatal("process context missing")
	}
	data, _ := json.Marshal(got)
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(data, &fields)
	if fields["process_context"] != nil || fields["before_xml"] != nil {
		t.Fatal("private input leaked")
	}
	pair.AfterXML = xml + " "
	if err := store.SaveBPMN(ctx, pair); err == nil {
		t.Fatal("immutable pair overwritten")
	}
	if _, err := store.GetBPMN(ctx, "../escape"); err == nil {
		t.Fatal("path traversal accepted")
	}
}
