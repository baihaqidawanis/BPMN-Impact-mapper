package textutil

import "testing"

func TestContextAnchorUsesSharedPhraseNotWholeNeighbor(t *testing.T) {
	got := ContextAnchor(
		"A required file is stored with the purchase request draft.",
		[]string{"Enter purchase request purpose and lines", "Validate request details"},
	)
	if got != "purchase request" {
		t.Fatalf("context anchor = %q", got)
	}
}

func TestContextAnchorFallsBackToSharedNeighborToken(t *testing.T) {
	got := ContextAnchor("", []string{"Prepare invoice", "Register invoice"})
	if got != "invoice" {
		t.Fatalf("context anchor = %q", got)
	}
	if got := ContextAnchor("A long unrelated instruction", []string{"Prepare invoice"}); got != "" {
		t.Fatalf("single unrelated neighbor leaked into anchor: %q", got)
	}
}
