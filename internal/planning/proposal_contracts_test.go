package planning

import (
	"context"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

func TestRelatedProposalContractsExcludeLocalAndNeverPromoteSourceClaims(t *testing.T) {
	local := FilePlan{ChangeType: domain.AddTask, ElementID: "A", File: "ui.ts", ProposedChanges: []string{"forward input"}}
	backend := FilePlan{ChangeType: domain.AddTask, ElementID: "A", File: "store.go", CurrentBehavior: "Unverified Planner assertion", EvidenceIDs: []string{"not-a-related-source-reference"}, TargetBehavior: "Validate before mutation", ProposedChanges: []string{"old guard"}, Assumptions: []string{"Confirm error contract"}}
	updated := backend
	updated.ProposedChanges = []string{"scoped guard"}
	contracts := BuildProposalContracts([]FilePlan{local, backend, updated}, []FilePlan{local})
	if len(contracts) != 1 || contracts[0].File != "store.go" || contracts[0].ProposedChanges[0] != "scoped guard" || contracts[0].TargetBehavior == "" {
		t.Fatal(contracts)
	}
	contracts[0].ProposedChanges[0] = "mutated"
	contracts[0].Assumptions[0] = "changed"
	if updated.ProposedChanges[0] != "scoped guard" || backend.Assumptions[0] != "Confirm error contract" {
		t.Fatal("related model input changed checkpoint plans")
	}
}

func TestFileReviewReceivesOtherContractsAndInvalidatesChangedContext(t *testing.T) {
	calls := map[string]int{}
	model := fileScopedReviewFixture{scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if strings.HasPrefix(turn.Feedback[0], "FINAL CONTRACT") {
			return AgentReply{Review: &AgentReview{Approved: true}}, nil
		}
		if len(turn.Draft) != 1 || len(turn.RelatedProposals) != 1 || turn.RelatedProposals[0].File == turn.Draft[0].File || turn.RelatedProposals[0].TargetBehavior == "" {
			t.Fatal("missing planned companion responsibility", turn.RelatedProposals)
		}
		calls[turn.Draft[0].File]++
		return AgentReply{Review: &AgentReview{Approved: true}}, nil
	}}}
	engine := FullRepositoryEngine{Model: model, Store: memorySessions{}}
	s := AgentSession{Binding: "related", Bundle: agentBundle(), Draft: []FilePlan{{File: "ui.ts", TargetBehavior: "Forward input", ProposedChanges: []string{"send input"}}, {File: "store.go", TargetBehavior: "Validate", ProposedChanges: []string{"guard input"}}}}
	for i := 0; i < 3; i++ {
		if i == 2 {
			s.Draft[1].ProposedChanges = []string{"revised scoped guard"}
		}
		part := s
		part.Draft = []FilePlan{s.Draft[0]}
		related := BuildProposalContracts(s.Draft, part.Draft)
		if _, _, err := engine.reviewConsistencyPart(context.Background(), "related-file-receipt", part, "FINAL FILE REVIEW", related); err != nil {
			t.Fatal(err)
		}
	}
	if calls["ui.ts"] != 2 {
		t.Fatal("changed companion proposal reused stale file approval", calls)
	}
}
