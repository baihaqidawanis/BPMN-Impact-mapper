package llm

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

type routedFixture struct {
	name       string
	version    string
	seen       []planning.AgentRole
	prepareErr error
	active     *atomic.Int32
	entered    chan struct{}
	unblock    chan struct{}
}

func (m *routedFixture) Prepare(context.Context) error { return m.prepareErr }
func (m *routedFixture) Identity() planning.ProviderIdentity {
	return planning.ProviderIdentity{Provider: "fixture", Model: m.name, PromptVersion: m.version}
}
func (m *routedFixture) Turn(ctx context.Context, turn planning.AgentTurn) (planning.AgentReply, error) {
	m.seen = append(m.seen, turn.Role)
	if m.active != nil {
		if m.active.Add(1) != 1 {
			return planning.AgentReply{}, errors.New("overlapping model turns")
		}
		defer m.active.Add(-1)
		close(m.entered)
		select {
		case <-m.unblock:
		case <-ctx.Done():
			return planning.AgentReply{}, ctx.Err()
		}
	}
	return planning.AgentReply{}, nil
}

func TestRoleRouterPinsEveryModelAndDoesNotFallback(t *testing.T) {
	a, p, v := &routedFixture{name: "a"}, &routedFixture{name: "p"}, &routedFixture{name: "v"}
	routes := map[planning.AgentRole]PreparedAgent{planning.Analyst: a, planning.Planner: p, planning.Reviewer: v}
	router, err := NewRoleRouter(routes)
	if err != nil {
		t.Fatal(err)
	}
	routes[planning.Analyst] = v
	if err = router.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, role := range agentRoles {
		if _, err = router.Turn(context.Background(), planning.AgentTurn{Role: role}); err != nil {
			t.Fatal(err)
		}
	}
	if len(a.seen) != 1 || a.seen[0] != planning.Analyst || len(p.seen) != 1 || p.seen[0] != planning.Planner || len(v.seen) != 1 || v.seen[0] != planning.Reviewer {
		t.Fatal("wrong role dispatch")
	}
	before := router.Identity()
	p.version = "changed"
	if router.Identity() == before {
		t.Fatal("planner identity drift omitted from session binding")
	}
	if _, err = router.Turn(context.Background(), planning.AgentTurn{Role: "coder"}); err == nil {
		t.Fatal("unknown role silently routed")
	}
	p.prepareErr = errors.New("model unavailable")
	if err = router.Prepare(context.Background()); err == nil {
		t.Fatal("unavailable role silently replaced")
	}
	delete(routes, planning.Reviewer)
	if _, err = NewRoleRouter(routes); err == nil {
		t.Fatal("incomplete routes accepted")
	}
}

func TestRoleRouterSerializesCallsAndHonorsCancellation(t *testing.T) {
	var active atomic.Int32
	f := &routedFixture{name: "one", active: &active, entered: make(chan struct{}), unblock: make(chan struct{})}
	r, _ := NewRoleRouter(map[planning.AgentRole]PreparedAgent{planning.Analyst: f, planning.Planner: f, planning.Reviewer: f})
	done := make(chan error, 1)
	go func() {
		_, err := r.Turn(context.Background(), planning.AgentTurn{Role: planning.Analyst})
		done <- err
	}()
	<-f.entered
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.Turn(ctx, planning.AgentTurn{Role: planning.Planner}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	close(f.unblock)
	if err := <-done; err != nil || active.Load() != 0 || len(f.seen) != 1 {
		t.Fatal("overlap or cancelled request ran", err)
	}
}

func TestOllamaRoleThinkingOverrideIsIndependentAndIdentityBound(t *testing.T) {
	base := OllamaAgentConfig{BaseURL: "http://127.0.0.1:11435", Model: "local-fixture", Profile: "coding"}
	plain, err := NewOllamaRoleRouter(base, nil)
	if err != nil {
		t.Fatal(err)
	}
	enabled := true
	router, err := NewOllamaRoleRouter(base, map[planning.AgentRole]OllamaRoleOverride{planning.Reviewer: {Think: &enabled}})
	if err != nil {
		t.Fatal(err)
	}
	if router.models[planning.Analyst] != router.models[planning.Planner] || router.models[planning.Reviewer] == router.models[planning.Analyst] || !router.models[planning.Reviewer].(*OllamaAgent).think || router.models[planning.Analyst].(*OllamaAgent).think {
		t.Fatal("thinking-only override leaked or was discarded")
	}
	if router.RoleIdentities()[planning.Analyst] != plain.RoleIdentities()[planning.Analyst] || router.Identity() == plain.Identity() {
		t.Fatal("role settings not reflected correctly in identity")
	}
	if router.PreferFileReview() {
		t.Fatal("thinking alone silently changed review partition policy")
	}
	jsonBase := base
	jsonBase.ReviewJSONMode = true
	scoped, err := NewOllamaRoleRouter(jsonBase, map[planning.AgentRole]OllamaRoleOverride{planning.Reviewer: {Think: &enabled}})
	if err != nil || !scoped.PreferFileReview() || scoped.models[planning.Analyst].(*OllamaAgent).think {
		t.Fatal("explicit review partition policy or independent thinking lost", err)
	}
	base.Think = true
	disabled := false
	router, err = NewOllamaRoleRouter(base, map[planning.AgentRole]OllamaRoleOverride{planning.Reviewer: {Think: &disabled}})
	if err != nil || router.models[planning.Reviewer].(*OllamaAgent).think || !router.models[planning.Planner].(*OllamaAgent).think {
		t.Fatal("explicit false must override inherited true", err)
	}
	if _, err := NewOllamaRoleRouter(base, map[planning.AgentRole]OllamaRoleOverride{"coder": {Model: "other"}}); err == nil {
		t.Fatal("unknown role accepted")
	}
	if _, err := NewOllamaRoleRouter(base, map[planning.AgentRole]OllamaRoleOverride{planning.Reviewer: {Model: "other:cloud"}}); err == nil {
		t.Fatal("unsafe override accepted")
	}
}
