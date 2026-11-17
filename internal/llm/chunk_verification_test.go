package llm

import (
	"encoding/json"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
	"strings"
	"testing"
)

func TestCompactChunkDecisionsBindExactSnapshotTargets(t *testing.T) {
	turn := planning.AgentTurn{RepositoryChunk: &planning.RepositoryChunk{File: "store.go"}}
	for _, id := range []string{"A", "B"} {
		turn.Bundle.Changes = append(turn.Bundle.Changes, planning.ChangeInput{ChangeType: domain.AddTask, ElementID: id, Candidates: []planning.Candidate{{File: "store.go", Evidence: []planning.EvidenceItem{{ID: "E" + id, Type: "FULL_SOURCE_CHUNK"}}}}})
	}
	valid := `{"verification":{"decisions":[{"change_index":1,"verdict":"REJECT","rationale":"Unrelated chunk"},{"change_index":0,"verdict":"WEAK","rationale":"Needs more source"}]}}`
	d := json.NewDecoder(strings.NewReader(valid))
	d.DisallowUnknownFields()
	out, err := decodeChunkVerification(d, turn)
	if err != nil {
		t.Fatal(err)
	}
	if out.Verification.Decisions[0].ElementID != "B" || out.Verification.Decisions[0].File != "store.go" || out.Verification.Decisions[0].EvidenceIDs[0] != "EB" {
		t.Fatal("snapshot identity lost")
	}
	for _, bad := range []string{
		strings.Replace(valid, `"change_index":1`, `"change_index":0`, 1),
		strings.Replace(valid, `"change_index":1`, `"change_index":2`, 1),
		strings.Replace(valid, `"change_index":1,`, "", 1),
		strings.Replace(valid, `"verdict":"REJECT"`, `"verdict":"CERTAIN"`, 1),
		strings.Replace(valid, `"change_index":1`, `"file":"invented.go","change_index":1`, 1),
		`{"verification":{"decisions":[]}}`,
	} {
		d = json.NewDecoder(strings.NewReader(bad))
		d.DisallowUnknownFields()
		if _, err := decodeChunkVerification(d, turn); err == nil {
			t.Fatal("invalid model decision accepted", bad)
		}
	}
	turn.RepositoryChunk.File = "other.go"
	d = json.NewDecoder(strings.NewReader(valid))
	d.DisallowUnknownFields()
	if _, err := decodeChunkVerification(d, turn); err == nil {
		t.Fatal("foreign chunk candidate accepted")
	}
}
