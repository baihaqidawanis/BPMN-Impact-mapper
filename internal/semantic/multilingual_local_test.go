package semantic_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/semantic"
)

// Opt-in diagnostic using authored English code snippets and equivalent English
// and Indonesian queries. It never downloads models. These six toy queries are
// engineering controls, not held-out accuracy evidence for real repositories.
func TestLocalBilingualRetrievalSmoke(t *testing.T) {
	if os.Getenv("BPMN_IMPACT_LOCAL_MODEL_SMOKE") != "1" {
		t.Skip("set BPMN_IMPACT_LOCAL_MODEL_SMOKE=1 with both models cached locally")
	}
	artifacts := []domain.Artifact{
		{File: "expense/receipt.go", Content: "// AttachReceipt stores the receipt file as supporting evidence for an employee reimbursement claim.\nfunc AttachReceipt(claimID string, receipt []byte) error { return storage.Save(claimID, receipt) }"},
		{File: "warehouse/adjustment.go", Content: "// CorrectStock updates warehouse inventory after a physical count and records an adjustment movement.\nfunc CorrectStock(productID string, countedQuantity int) error { return inventory.Adjust(productID, countedQuantity) }"},
		{File: "master/supplier.go", Content: "// DeactivateSupplier disables a supplier so it cannot be selected for new purchasing documents.\nfunc DeactivateSupplier(supplierID string) error { return suppliers.SetActive(supplierID, false) }"},
		{File: "auth/login.tsx", Content: "// Sign in to the business workspace. Purchases, warehouse stock, and finance in one application.\nexport function Login() { return <form><input name='password'/><button>Sign in</button></form> }"},
		{File: "README.md", Content: "Install the application and configure the database. This business workspace manages purchases, inventory, employee expenses, and supplier payments. Run tests before deploying."},
		{File: "sales/invoice.go", Content: "// IssueInvoice creates a customer invoice after shipment.\nfunc IssueInvoice(orderID string) error { return invoices.Create(orderID) }"},
	}
	cases := []struct{ query, expected string }{
		{"Attach receipt to reimbursement claim", "expense/receipt.go"},
		{"Lampirkan kuitansi pada klaim penggantian biaya", "expense/receipt.go"},
		{"Adjust warehouse stock after a physical count", "warehouse/adjustment.go"},
		{"Sesuaikan stok gudang setelah perhitungan fisik", "warehouse/adjustment.go"},
		{"Deactivate a supplier", "master/supplier.go"},
		{"Nonaktifkan pemasok", "master/supplier.go"},
	}
	changes := make([]domain.Change, len(cases))
	for i, c := range cases {
		changes[i] = domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{Name: c.query, Type: "userTask"}}
	}
	for _, model := range []string{semantic.DefaultModel, "intfloat/multilingual-e5-small"} {
		t.Run(model, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			embedder := semantic.PythonEmbedder{Command: "python", Model: model}
			result, _, err := (semantic.Strategy{Embedder: embedder, DiversifyFiles: true}).Attach(ctx, changes, artifacts, mapping.Options{TopK: len(artifacts)})
			if err != nil {
				t.Fatal(err)
			}
			top1, top3 := 0, 0
			for i, c := range cases {
				rank, first := 0, "unresolved"
				for j, impact := range result[i].Impacts {
					if j == 0 {
						first = impact.File
					}
					if impact.File == c.expected {
						rank = j + 1
					}
				}
				if rank == 1 {
					top1++
				}
				if rank > 0 && rank <= 3 {
					top3++
				}
				t.Logf("query=%q expected_rank=%d first=%s", c.query, rank, first)
			}
			t.Logf("synthetic controls only: top1=%d/%d top3=%d/%d", top1, len(cases), top3, len(cases))
		})
	}
}
