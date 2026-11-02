package semantic

import (
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

func TestAdjacentPassagesCoverHeadGapWithoutQueryOrUnboundedInput(t *testing.T) {
	content := strings.Repeat("a", 800) + "GUARD_MARKER" + strings.Repeat("b", 3300)
	a := domain.Artifact{Content: content}
	plain, expanded := artifactPassages(a), artifactPassagesWithAdjacent(a, true)
	for _, p := range plain {
		if strings.Contains(p.content, "GUARD_MARKER") {
			t.Fatal("fixture marker not in old sampling gap")
		}
	}
	found := false
	for _, p := range expanded {
		if len([]rune(p.content)) > maxPassageCharacters {
			t.Fatal("unbounded passage")
		}
		found = found || strings.Contains(p.content, "GUARD_MARKER")
	}
	if !found || len(expanded) > 5 {
		t.Fatalf("coverage/limit: %d", len(expanded))
	}
	if len(artifactPassagesWithAdjacent(domain.Artifact{Content: "short"}, true)) != 1 {
		t.Fatal("duplicate short passage")
	}
}
