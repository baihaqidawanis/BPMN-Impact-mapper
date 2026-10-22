package retrieval

import (
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"strconv"
	"strings"
	"testing"
)

func TestStructuredEvidenceRequiresRelevantSyntaxOnReportedLine(t *testing.T) {
	cases := []struct {
		name, content, kind string
		line                int
	}{
		{"trace alone", `trace := "PUR-PR-002"`, "", 0},
		{"plain update text", `update purchase request details`, "", 0},
		{"SQL update", `UPDATE purchase_requests SET status = 'DRAFT'`, "DATABASE_TABLE", 10},
		{"unrelated SQL", "trace := \"purchase\"\nSELECT id FROM invoices", "", 0},
		{"unrelated table but related column", "SELECT request_id FROM customers WHERE purchase_request_id=$1", "", 0},
		{"unrelated table but related value", "UPDATE customers SET note='purchase request'", "", 0},
		{"SQL location", "trace := \"purchase\"\nSELECT id FROM purchase_requests", "DATABASE_TABLE", 11},
		{"select channel", "select {\ncase purchase := <-requests:", "", 0},
		{"uppercase ID", `id := "PURCHASE-001"`, "", 0},
		{"test body is not test name", `purchase := "request"`, "", 0},
		{"test declaration", `func TestPurchaseRequest(t *testing.T) {`, "TEST_NAME", 10},
		{"unrelated test body", `func TestInvoices(t *testing.T) { request := 1 }`, "", 0},
		{"route", `router.Post("/purchase-requests", handler)`, "API_ROUTE", 10},
		{"unrelated route argument", `router.Post("/orders", purchaseRequest)`, "", 0},
		{"status", `purchaseStatus := "DRAFT"`, "STATE_STATUS", 10},
		{"unrelated status neighbour", `return Document{DocumentNo: "request", Status: "DRAFT"}`, "", 0},
		{"audit call", `audit(ctx, "CREATE_PURCHASE_REQUEST")`, "AUDIT_ACTION", 10},
		{"unrelated audit argument", `audit(ctx, "CREATE_SALES_ORDER", request)`, "", 0},
		{"comment", `// SELECT id FROM purchase_requests`, "", 0},
		{"clipped syntax", strings.Repeat(" ", 1) + strings.Repeat("x", 245) + ` SELECT id FROM purchase_requests`, "", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			score, e := structuredSignal(domain.Artifact{File: "test/store.go", Layer: "test", Content: tc.content, Symbol: domain.Symbol{StartLine: 10}}, []string{"purchase"})
			if e.Type != tc.kind {
				t.Fatalf("got %s evidence=%+v", e.Type, e)
			}
			if tc.kind == "" && score != 0 {
				t.Fatalf("unexpected score %v", score)
			}
			if tc.line > 0 {
				if e.Location != "test/store.go:"+strconv.Itoa(tc.line) {
					t.Fatalf("wrong location: %+v", e)
				}
				if !strings.Contains(tc.content, e.Excerpt) {
					t.Fatalf("not a source excerpt: %+v", e)
				}
			}
		})
	}
}

func TestDatabaseTableStrengthUsesIdentifierCoverage(t *testing.T) {
	query := []string{"purchase", "request", "upload", "document"}
	cases := []struct {
		sql  string
		want float64
	}{
		{`UPDATE purchase_requests SET status='DRAFT'`, 1},
		{`UPDATE purchase_orders SET status='DRAFT'`, 0.5},
		{`UPDATE products SET note='purchase request'`, 0},
	}
	for _, tc := range cases {
		got, _ := structuredSignal(domain.Artifact{File: "store.go", Content: tc.sql}, query)
		if got != tc.want {
			t.Fatalf("SQL %q: got %v want %v", tc.sql, got, tc.want)
		}
	}
}
