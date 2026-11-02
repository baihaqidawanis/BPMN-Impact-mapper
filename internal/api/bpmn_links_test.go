package api_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/api"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmnlib"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/filestore"
)

func TestLibraryHierarchyValidationAndPersistence(t *testing.T) {
	store := filestore.BPMNLibrary{Root: t.TempDir()}
	xml := strings.ReplaceAll(libraryXML, "exclusiveGateway", "userTask")
	makeEntry := func(repo string, level int, version string) bpmnlib.Entry {
		e, err := bpmnlib.Prepare(bpmnlib.Metadata{Repository: repo, Process: "Process", Level: level, Version: version}, xml)
		if err != nil {
			t.Fatal(err)
		}
		if err = store.Put(e); err != nil {
			t.Fatal(err)
		}
		return e
	}
	parent := makeEntry("Demo", 2, "1")
	child := makeEntry("Demo", 3, "1")
	other := makeEntry("Other", 3, "1")
	call := func(parentID, element, target string) int {
		data, _ := json.Marshal(map[string]string{"element_id": element, "target_id": target})
		w := httptest.NewRecorder()
		api.Handler{Library: store}.ServeHTTP(w, httptest.NewRequest("PUT", "/api/v1/bpmn-library/"+parentID+"/links", bytes.NewReader(data)))
		return w.Code
	}
	for _, c := range []struct {
		parent, element, target string
		status                  int
	}{
		{parent.ID, "s", child.ID, 400}, // events cannot be decomposition parents
		{parent.ID, "g", other.ID, 400}, // repository mismatch
		{parent.ID, "g", parent.ID, 400},
		{child.ID, "g", parent.ID, 400}, // prevents backward links/cycles
		{parent.ID, "g", "missing", 404},
		{parent.ID, "g", child.ID, 200},
	} {
		if got := call(c.parent, c.element, c.target); got != c.status {
			t.Fatalf("link %+v: status %d", c, got)
		}
	}
	reloaded := filestore.BPMNLibrary{Root: store.Root}
	links, err := reloaded.Links(parent.ID)
	if err != nil || len(links) != 1 || links[0].TargetID != child.ID {
		t.Fatalf("lost persisted link: %+v %v", links, err)
	}
	version2 := makeEntry("Demo", 3, "2")
	if call(parent.ID, "g", version2.ID) != 200 {
		t.Fatal("relink failed")
	}
	links, _ = reloaded.Links(parent.ID)
	if len(links) != 1 || links[0].TargetID != version2.ID {
		t.Fatal("relink appended instead of replacing")
	}
	if call(parent.ID, "g", "") != 200 {
		t.Fatal("unlink failed")
	}
	links, _ = reloaded.Links(parent.ID)
	if len(links) != 0 {
		t.Fatal("unlink not persisted")
	}
	unchanged, _ := store.Get(parent.ID)
	if unchanged.XML != xml {
		t.Fatal("link changed parent XML")
	}
}
