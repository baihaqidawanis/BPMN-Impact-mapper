package planning

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

func TestNumberedMigrationIsEvidenceWithoutEditingHistory(t *testing.T) {
	file := "backend/migrations/000005_operations.up.sql"
	for _, action := range []domain.RequiredAction{domain.ActionModify, domain.ActionRemove} {
		err := validateMigrationEdit(FilePlan{File: file, Action: action})
		if !errors.Is(err, ErrAgentPlanIncomplete) || !strings.Contains(err.Error(), "new forward migration") {
			t.Fatalf("expected actionable repair for %s: %v", action, err)
		}
	}
	for _, plan := range []FilePlan{
		{File: file, Action: domain.ActionNoAction},
		{File: "backend/migrations/000006_next.up.sql", Action: domain.ActionAdd},
		{File: "backend/schema.sql", Action: domain.ActionModify},
	} {
		if err := validateMigrationEdit(plan); err != nil {
			t.Fatalf("rejected %s for %s: %v", plan.Action, plan.File, err)
		}
	}
}

func TestRoutingGuardRejectsExplicitBusinessContradictions(t *testing.T) {
	change := ChangeInput{BranchChanges: []domain.BranchChange{{NewCondition: "semua selisih nol"}}}
	for _, step := range []string{
		"Calculate the sum of deltas for all lines before choosing the zero path.",
		"If the calculated delta sum is zero, set terminal status.",
		"Return success without writing the new status.",
		"Set d.Status = \"NO_ADJUSTMENT\" and return immediately without executing the stock mutation logic.",
	} {
		plan := FilePlan{File: "store.go", Steps: []string{step}}
		if err := validateRoutingPlan(change, plan); !errors.Is(err, ErrAgentPlanIncomplete) {
			t.Errorf("accepted conflicting step %q: %v", step, err)
		}
	}
	for _, step := range []string{
		"Compare the snapshot and counted quantity for every line, then persist the selected status and commit.",
		"Return HTTP 409 without writing the new status when the version is stale.",
		"Set d.Status = \"NO_ADJUSTMENT\", persist it, and return immediately after commit.",
		"Use a count of nonzero differences to select the zero path.",
	} {
		plan := FilePlan{File: "store.go", ProposedChanges: []string{step}}
		if err := validateRoutingPlan(change, plan); err != nil {
			t.Errorf("rejected valid step %q: %v", step, err)
		}
	}
	aggregate := ChangeInput{BranchChanges: []domain.BranchChange{{NewCondition: "net balance == 0"}}}
	if err := validateRoutingPlan(aggregate, FilePlan{Steps: []string{"Calculate the sum of deltas."}}); err != nil {
		t.Fatal("a true aggregate requirement may use an aggregate", err)
	}
}

func TestItemLevelTaskAlsoValidatesProposedTestExpectations(t *testing.T) {
	change := ChangeInput{After: &ElementInput{Name: "Periksa apakah semua selisih stok nol"}}
	plan := FilePlan{Layer: "test", AcceptanceChecks: []string{"If the sum of deltas is zero, return success."}}
	if err := validateRoutingPlan(change, plan); !errors.Is(err, ErrAgentPlanIncomplete) {
		t.Fatal("accepted cancellation in an item-level task's acceptance check", err)
	}
	plan.AcceptanceChecks = []string{"Do not calculate the sum of deltas. Compare each row with its snapshot."}
	if err := validateRoutingPlan(change, plan); err != nil {
		t.Fatal(err)
	}
}

func TestRoutingGuardCatchesMidSentenceAggregatesRouteInversionAndSelfContradiction(t *testing.T) {
	pc := &domain.ProcessContext{
		After: &domain.Graph{
			Elements: map[string]*domain.Element{
				"ZeroCheck": {ID: "ZeroCheck", Type: "exclusiveGateway", Name: "Semua selisih nol?"},
				"CloseZero": {ID: "CloseZero", Type: "serviceTask", Name: "Tutup sebagai NO_ADJUSTMENT", Documentation: "Tanpa posting stok; posting stok tetap aksi terpisah."},
				"NormalSub": {ID: "NormalSub", Type: "serviceTask", Name: "Set status SUBMITTED & antre approval"},
			},
			Flows: map[string]*domain.Flow{
				"fZero":    {ID: "fZero", SourceRef: "ZeroCheck", TargetRef: "CloseZero", Condition: "semua selisih nol"},
				"fNonZero": {ID: "fNonZero", SourceRef: "ZeroCheck", TargetRef: "NormalSub", Condition: "ada baris dengan selisih nonnol"},
			},
		},
	}
	// Adjacent task/join change does not repeat the gateway condition in change.After.
	adjacent := ChangeInput{ChangeType: domain.AddTask, ElementID: "JoinOutcome", After: &ElementInput{Name: "Gabungkan hasil"}}
	for _, bad := range []string{
		"In backend/store.go, within TransitionAdjustment, add logic to query stock_movements and sum the quantity differences against the snapshot.",
		"In backend/store.go, query the adjustment lines and calculate the sum of quantity differences against the snapshot.",
		"Introduce a conditional check: if the sum of deltas is non-zero, return a specific validation error.",
		"In backend/store.go, add logic to calculate the total delta between the requested adjustment lines and the current snapshot.",
		"If any line item has a non-zero delta, reject with a validation error.",
		"Submit an adjustment where line-item deltas sum to zero; expect NO_ADJUSTMENT.",
		"Introduce a condition to check if the sum of all line deltas is zero; if true, set NO_ADJUSTMENT.",
		"Add a conditional branch: if all line deltas are zero, return success immediately without updating d.Status.",
		"Submit an adjustment with a non-zero delta while in DRAFT status; expect the response to indicate validation failure.",
		"If any line delta is non-zero, proceed to set d.Status = \"SUBMITTED\" and execute the stock update query.",
		"Submit with non-zero delta: Status becomes SUBMITTED, audit records APPROVED, stock posting occurs.",
	} {
		plan := FilePlan{File: "backend/store.go", Steps: []string{bad}}
		err := validateRoutingPlanWithContext(pc, adjacent, plan)
		if !errors.Is(err, ErrAgentPlanIncomplete) || !strings.Contains(err.Error(), "offending text:") {
			t.Errorf("expected offending-text error for %q, got %v", bad, err)
		}
	}
	// Rejecting zero-delta with an error inverts the bypass branch in after BPMN.
	zeroReject := FilePlan{
		File:             "backend/adjustment_test.go",
		Layer:            "test",
		AcceptanceChecks: []string{"Verify that calling Submit() on an adjustment with zero delta returns a non-nil error."},
	}
	if err := validateRoutingPlanWithContext(pc, adjacent, zeroReject); !errors.Is(err, ErrAgentPlanIncomplete) {
		t.Fatalf("accepted zero-delta rejection inversion: %v", err)
	}
	// Assigning SUBMITTED or keeping DRAFT on the zero-delta branch contradicts the after target status NO_ADJUSTMENT.
	for _, badCheck := range []string{
		"Submit with zero delta: Status becomes SUBMITTED, version increments, audit records NO_ADJUSTMENT.",
		"Submit with all line deltas zero: Status remains DRAFT, no SUBMITTED transition, success response returned.",
		"Submit a DRAFT adjustment where all line deltas are zero; the system returns success, status remains SUBMITTED, version increments, and no stock_movements rows are created.",
	} {
		wrongZeroStatus := FilePlan{File: "backend/store.go", AcceptanceChecks: []string{badCheck}}
		if err := validateRoutingPlanWithContext(pc, adjacent, wrongZeroStatus); !errors.Is(err, ErrAgentPlanIncomplete) {
			t.Fatalf("accepted wrong zero-delta status %q: %v", badCheck, err)
		}
	}
	// Repurposing an existing Approval test into a zero-delta bypass test breaks regression coverage.
	repurposed := FilePlan{
		File:  "backend/test/integration/operations_test.go",
		Layer: "test",
		Steps: []string{"Update the test TestInventoryApprovalRejectsChangedSnapshot to construct zero-delta lines and assert NO_ADJUSTMENT rather than SUBMITTED."},
	}
	if err := validateRoutingPlanWithContext(pc, adjacent, repurposed); !errors.Is(err, ErrAgentPlanIncomplete) {
		t.Fatalf("accepted repurposed Approval regression test: %v", err)
	}
	// Lowercase "return success" must NOT collide with uppercase status literals (NO_ADJUSTMENT vs success bug).
	withReturnSuccess := FilePlan{
		File:             "backend/store.go",
		Steps:            []string{"If all lines have zero delta, set status to NO_ADJUSTMENT and return success after commit."},
		AcceptanceChecks: []string{"When all lines have zero delta, return success and status NO_ADJUSTMENT."},
	}
	if err := validateRoutingPlanWithContext(pc, adjacent, withReturnSuccess); err != nil {
		t.Fatalf("false positive on lowercase 'return success': %v", err)
	}
	selfConflict := FilePlan{
		File:             "backend/store.go",
		Steps:            []string{"If all lines have zero delta, set status to NO_ADJUSTMENT and commit."},
		AcceptanceChecks: []string{"When submitting a DRAFT adjustment with zero delta per line, the function returns SUBMITTED status."},
	}
	if err := validateRoutingPlanWithContext(pc, adjacent, selfConflict); !errors.Is(err, ErrAgentPlanIncomplete) {
		t.Fatalf("accepted self-contradictory zero-delta proposal: %v", err)
	}
	valid := FilePlan{
		File: "backend/store.go",
		Steps: []string{
			"Compare each line against its snapshot; do not calculate the sum of deltas.",
			"If all lines have zero delta, set status to NO_ADJUSTMENT; otherwise transition to SUBMITTED.",
		},
		AcceptanceChecks: []string{
			"Even when the total delta is zero (+2 and -2), transition to SUBMITTED.",
			"When all lines have zero delta, set status to NO_ADJUSTMENT without mutating stock.",
		},
	}
	if err := validateRoutingPlanWithContext(pc, adjacent, valid); err != nil {
		t.Fatalf("rejected valid per-line plan: %v", err)
	}
}

func TestMigrationPrerequisiteFlagsMissingForwardMigrationForNewStatus(t *testing.T) {
	bundle := EvidenceBundle{
		ProcessContext: &domain.ProcessContext{
			Before: &domain.Graph{Elements: map[string]*domain.Element{"sub": {Name: "Submit as SUBMITTED"}}},
			After: &domain.Graph{Elements: map[string]*domain.Element{
				"sub":   {Name: "Submit as SUBMITTED"},
				"close": {Name: "Close without mutation (NO_ADJUSTMENT)"},
			}},
		},
		Changes: []ChangeInput{{
			ChangeType: domain.AddGateway,
			ElementID:  "ZeroCheck",
			Candidates: []Candidate{{
				File: "backend/migrations/000005_operations.up.sql",
				Evidence: []EvidenceItem{{
					ID:      "sql-1",
					Type:    "SOURCE_TOOL",
					Excerpt: "status TEXT NOT NULL CHECK(status IN('DRAFT','SUBMITTED','APPROVED','REJECTED'))",
				}, {
					ID:      "sql-2",
					Type:    "SOURCE_TOOL",
					Excerpt: "movement_type TEXT NOT NULL CHECK(movement_type IN('STOCK_ADJUSTMENT','SALES_DELIVERY'))",
				}},
			}},
		}},
	}
	historicalNoAction := FilePlan{
		File:            "backend/migrations/000005_operations.up.sql",
		Action:          domain.ActionNoAction,
		Summary:         "Historical migration cannot be edited; a new forward migration is required.",
		TargetBehavior:  "Preserve historical migration.",
		DesignRationale: "Do not mutate applied DDL.",
	}
	storageWithoutMigration := FilePlan{
		File:            "backend/store.go",
		Layer:           "adapter",
		Action:          domain.ActionModify,
		TargetBehavior:  "Persist status NO_ADJUSTMENT when all lines have zero delta.",
		ProposedChanges: []string{"Set adjustment status to NO_ADJUSTMENT and commit."},
	}
	if err := validateMigrationPrerequisites(bundle, []FilePlan{historicalNoAction, storageWithoutMigration}); !errors.Is(err, ErrAgentPlanIncomplete) {
		t.Fatalf("expected missing forward migration error, got %v", err)
	}
	wrongMigration := storageWithoutMigration
	wrongMigration.Steps = []string{"Add a new forward migration to allow NO_ADJUSTMENT in the movement_type CHECK constraint."}
	issues := migrationPrerequisiteIssues(bundle, []FilePlan{historicalNoAction, wrongMigration})
	if len(issues) != 1 || !strings.Contains(issues[0], "restricts status") {
		t.Fatalf("expected only a warning for status despite unrelated migration, got %v", issues)
	}
	movementOnly := bundle
	movementOnly.Changes = append([]ChangeInput(nil), bundle.Changes...)
	movementOnly.Changes[0].Candidates = append([]Candidate(nil), bundle.Changes[0].Candidates...)
	movementOnly.Changes[0].Candidates[0].Evidence = bundle.Changes[0].Candidates[0].Evidence[1:]
	if issues := migrationPrerequisiteIssues(movementOnly, []FilePlan{storageWithoutMigration}); len(issues) != 0 {
		t.Fatalf("movement evidence alone cannot establish a status constraint, got %v", issues)
	}
	storageWithMigration := storageWithoutMigration
	storageWithMigration.Steps = append(storageWithMigration.Steps, "Add a new forward migration to update the status CHECK constraint to allow NO_ADJUSTMENT, with a matching .down.sql rollback strategy handling existing NO_ADJUSTMENT rows.")
	if err := validateMigrationPrerequisites(bundle, []FilePlan{historicalNoAction, storageWithMigration}); err != nil {
		t.Fatalf("rejected storage proposal with explicit forward migration: %v", err)
	}
	missingRollback := storageWithoutMigration
	missingRollback.Steps = append(missingRollback.Steps, "Add a new forward migration to update the status CHECK constraint to allow NO_ADJUSTMENT.")
	if issues := migrationPrerequisiteIssues(bundle, []FilePlan{historicalNoAction, missingRollback}); len(issues) == 0 || !strings.Contains(strings.Join(issues, "\n"), "rollback (.down.sql) strategy") {
		t.Fatalf("expected missing rollback strategy warning, got: %v", issues)
	}
	dbMigrationPhrasing := storageWithoutMigration
	dbMigrationPhrasing.Steps = append(dbMigrationPhrasing.Steps, "Apply database migration updating the status CHECK constraint defined in backend/migrations/000005_operations.up.sql to allow 'NO_ADJUSTMENT', and document the rollback strategy for existing rows.")
	if err := validateMigrationPrerequisites(bundle, []FilePlan{historicalNoAction, dbMigrationPhrasing}); err != nil {
		t.Fatalf("rejected valid database migration phrasing referencing baseline file: %v", err)
	}
	proseMutatesHistorical := storageWithoutMigration
	proseMutatesHistorical.ProposedChanges = append([]string{"Update `backend/migrations/000005_operations.up.sql` to add 'NO_ADJUSTMENT' to the status CHECK constraint."}, storageWithoutMigration.ProposedChanges...)
	if err := validateMigrationPrerequisites(bundle, []FilePlan{historicalNoAction, proseMutatesHistorical}); !errors.Is(err, ErrAgentPlanIncomplete) {
		t.Fatalf("expected error when prose mutates historical migration file directly, got %v", err)
	}

	// Unrelated domain migrations (.down.sql or finance/sales tables) and pure /domain/ files must not block inventory storage.
	crossDomainBundle := bundle
	crossDomainBundle.ProcessContext = &domain.ProcessContext{
		Before: &domain.Graph{ProcessNames: map[string]string{"p1": "Submit stock adjustment"}, Elements: map[string]*domain.Element{"sub": {Name: "Submit as SUBMITTED"}}},
		After: &domain.Graph{ProcessNames: map[string]string{"p1": "Submit stock adjustment"}, Elements: map[string]*domain.Element{
			"sub":   {Name: "Submit as SUBMITTED"},
			"close": {Name: "Close without mutation (NO_ADJUSTMENT)"},
		}},
	}
	crossDomainBundle.Changes = []ChangeInput{{
		ChangeType: domain.AddGateway,
		ElementID:  "ZeroCheck",
		Candidates: []Candidate{
			{
				File: "backend/migrations/000006_finance.down.sql",
				Evidence: []EvidenceItem{{ID: "down-1", Type: "SOURCE_TOOL", Excerpt: "status TEXT NOT NULL CHECK(status IN('DRAFT','POSTED','VOID'))"}},
			},
			{
				File: "backend/migrations/000006_finance.up.sql",
				Evidence: []EvidenceItem{{ID: "fin-1", Type: "SOURCE_TOOL", Excerpt: "ALTER TABLE sales_orders ADD CONSTRAINT sales_orders_status_check CHECK(status IN('DRAFT','SUBMITTED','APPROVED')); CREATE TABLE supplier_invoices (status TEXT NOT NULL CHECK(status IN('POSTED','PAID','VOID')));"}},
			},
		},
	}}
	invStore := FilePlan{
		File:            "backend/internal/inventory/adapter/postgres/store.go",
		Layer:           "adapter",
		Action:          domain.ActionModify,
		TargetBehavior:  "Persist status NO_ADJUSTMENT when all lines have zero delta.",
		ProposedChanges: []string{"Set adjustment status to NO_ADJUSTMENT and commit."},
	}
	invDomain := FilePlan{
		File:            "backend/internal/inventory/domain/adjustment.go",
		Layer:           "domain",
		Action:          domain.ActionModify,
		TargetBehavior:  "Set status to NO_ADJUSTMENT when all lines have zero delta.",
		ProposedChanges: []string{"Set status to NO_ADJUSTMENT."},
	}
	if issues := migrationPrerequisiteIssues(crossDomainBundle, []FilePlan{invStore, invDomain}); len(issues) != 0 {
		t.Fatalf("unrelated finance/rollback constraints or domain struct triggered migration prerequisite error: %v", issues)
	}
	if issues := migrationPrerequisiteIssues(bundle, []FilePlan{invDomain}); len(issues) != 0 {
		t.Fatalf("pure domain struct should not own SQL DDL prerequisite: %v", issues)
	}

	// Migration number collision, exact table name from split SQL lines, and non-zero-only ledger table guard.
	splitSQLBundle := crossDomainBundle
	splitSQLBundle.ProcessContext.After.Elements["zero"] = &domain.Element{ID: "zero", Type: "serviceTask", Name: "Periksa apakah semua selisih stok nol"}
	splitSQLBundle.Changes = []ChangeInput{{
		ChangeType: domain.AddGateway,
		ElementID:  "ZeroDeltaDecision",
		Candidates: []Candidate{{
			File: "backend/migrations/000005_operations.up.sql",
			Evidence: []EvidenceItem{
				{ID: "s1", Type: "SOURCE_TOOL", Location: "backend/migrations/000005_operations.up.sql:36:1", Excerpt: "CREATE TABLE inventory_adjustments (id uuid PRIMARY KEY, document_no text"},
				{ID: "s2", Type: "SOURCE_TOOL", Location: "backend/migrations/000005_operations.up.sql:36:201", Excerpt: "status text NOT NULL CHECK(status IN('DRAFT','SUBMITTED','APPROVED','REJECTED'))"},
				{ID: "s3", Type: "SOURCE_TOOL", Location: "backend/migrations/000005_operations.up.sql:37:1", Excerpt: "CREATE TABLE inventory_adjustment_lines (id uuid PRIMARY KEY, snapshot_quantity numeric, counted_quantity numeric)"},
				{ID: "s3b", Type: "SOURCE_TOOL", Location: "backend/migrations/000005_operations.up.sql:39:1", Excerpt: "CREATE TABLE stock_movements (id uuid PRIMARY KEY"},
				{ID: "s4", Type: "SOURCE_TOOL", Location: "backend/migrations/000005_operations.up.sql:41:1", Excerpt: "quantity numeric(19,4) NOT NULL CHECK(quantity<>0), movement_type text NOT NULL CHECK(movement_type IN('STOCK_ADJUSTMENT','SALES_DELIVERY'))"},
			},
		}},
	}}
	relatedFinance := []ProposalContract{{File: "backend/migrations/000006_finance.up.sql", Action: domain.ActionNoAction}}
	badNumAndTable := FilePlan{
		File:            "backend/internal/inventory/adapter/postgres/store.go",
		Layer:           "adapter",
		Action:          domain.ActionModify,
		TargetBehavior:  "Set status to NO_ADJUSTMENT when all lines have zero delta.",
		ProposedChanges: []string{"Query stock_movements to compute line deltas and set status to NO_ADJUSTMENT."},
		Steps:           []string{"Add a new forward migration 000006_add_no_adjustment.up.sql altering the adjustments table status CHECK constraint to allow NO_ADJUSTMENT."},
	}
	round5Issues := migrationPrerequisiteIssuesWithRelated(splitSQLBundle, []FilePlan{badNumAndTable}, relatedFinance)
	joinedIssues := strings.Join(round5Issues, "\n")
	for _, want := range []string{"collides with existing migrations", "inventory_adjustment_lines", "exact SQL table \"inventory_adjustments\""} {
		if !strings.Contains(joinedIssues, want) {
			t.Fatalf("expected issue containing %q, got:\n%s", want, joinedIssues)
		}
	}
	cleanStore := FilePlan{
		File:            "backend/internal/inventory/adapter/postgres/store.go",
		Layer:           "adapter",
		Action:          domain.ActionModify,
		TargetBehavior:  "Set status to NO_ADJUSTMENT when all lines have zero delta.",
		ProposedChanges: []string{"Query inventory_adjustment_lines to compare counted_quantity and snapshot_quantity per line, then set status to NO_ADJUSTMENT when all deltas are zero."},
		Steps:           []string{"Add a new forward migration 000007_add_no_adjustment_status.up.sql altering inventory_adjustments status CHECK constraint to allow NO_ADJUSTMENT, plus a matching .down.sql rollback plan for existing NO_ADJUSTMENT rows."},
	}
	if issues := migrationPrerequisiteIssuesWithRelated(splitSQLBundle, []FilePlan{cleanStore}, relatedFinance); len(issues) != 0 {
		t.Fatalf("rejected clean storage proposal: %v", issues)
	}
	spacedLedger := cleanStore
	spacedLedger.CurrentBehavior = "TransitionAdjustment validates DRAFT status and version, then queries stock movements."
	spacedLedger.ProposedChanges = []string{"Modify TransitionAdjustment to query stock movements for the adjustment ID and set status to NO_ADJUSTMENT."}
	if issues := migrationPrerequisiteIssuesWithRelated(splitSQLBundle, []FilePlan{spacedLedger}, relatedFinance); len(issues) == 0 || !strings.Contains(strings.Join(issues, "\n"), "inventory_adjustment_lines") || !strings.Contains(strings.Join(issues, "\n"), "snapshot_quantity vs counted_quantity") {
		t.Fatalf("expected spaced 'stock movements' query to be caught with line columns, got: %v", issues)
	}
	payloadGuessStore := cleanStore
	payloadGuessStore.ProposedChanges = []string{"Calculate the delta for each line item in the request payload and set status to NO_ADJUSTMENT."}
	if issues := migrationPrerequisiteIssuesWithRelated(splitSQLBundle, []FilePlan{payloadGuessStore}, relatedFinance); len(issues) == 0 || !strings.Contains(strings.Join(issues, "\n"), "snapshot_quantity vs counted_quantity") {
		t.Fatalf("expected request-payload delta hallucination to be caught with line quantity columns, got: %v", issues)
	}

	// Uncalled /domain/ method proved empty by find_callers must not be modified.
	uncalledObs := []ToolObservation{{
		Request: ToolRequest{Name: "find_callers", File: "backend/internal/inventory/domain/adjustment.go", Symbol: "Adjustment.Complete"},
		Result:  ToolResult{Candidates: []Candidate{}},
	}}
	if err := validateUncalledDomainMethodEdit(uncalledObs, FilePlan{
		File: "backend/internal/inventory/domain/adjustment.go", Symbol: "Adjustment.Complete", Layer: "domain", Action: domain.ActionModify,
	}); !errors.Is(err, ErrAgentPlanIncomplete) {
		t.Fatalf("expected uncalled domain method edit to be rejected, got %v", err)
	}

	// Deferred stock posting is valid (0 false positive); active stock_balances insert or +2/-2 -> NO_ADJUSTMENT is rejected.
	skipPC := &domain.ProcessContext{
		After: &domain.Graph{
			Elements: map[string]*domain.Element{
				"Zero":  {ID: "Zero", Type: "exclusiveGateway", Name: "Semua selisih nol?"},
				"Close": {ID: "Close", Type: "serviceTask", Name: "Tutup NO_ADJUSTMENT", Documentation: "Lewati antrean approval; posting stok tetap aksi terpisah."},
				"Sub":   {ID: "Sub", Type: "serviceTask", Name: "Ubah adjustment SUBMITTED"},
			},
			Flows: map[string]*domain.Flow{
				"f1": {SourceRef: "Zero", TargetRef: "Close", Condition: "semua selisih nol"},
				"f2": {SourceRef: "Zero", TargetRef: "Sub", Condition: "ada baris dengan selisih nonnol"},
			},
		},
	}
	deferredOK := FilePlan{
		File:             "backend/internal/inventory/adapter/postgres/store.go",
		Layer:            "adapter",
		Action:           domain.ActionModify,
		ProposedChanges:  []string{"Compare counted_quantity vs snapshot_quantity per line."},
		AcceptanceChecks: []string{"Submit a DRAFT adjustment with at least one non-zero delta: status becomes SUBMITTED, and stock posting is deferred until after approval.", "Submit a DRAFT adjustment with non-zero deltas; expect standard approval flow (status SUBMITTED) and stock posting deferred."},
	}
	if err := validateRoutingPlanWithContext(skipPC, ChangeInput{}, deferredOK); err != nil {
		t.Fatalf("false positive on deferred stock posting: %v", err)
	}
	prematureInsert := deferredOK
	prematureInsert.AcceptanceChecks = []string{"Verify non-zero deltas follow SUBMITTED status and inserts into stock_balances."}
	if err := validateRoutingPlanWithContext(skipPC, ChangeInput{}, prematureInsert); !errors.Is(err, ErrAgentPlanIncomplete) {
		t.Fatalf("expected premature stock_balances insert on submit to fail, got %v", err)
	}
	prematureQueued := deferredOK
	prematureQueued.AcceptanceChecks = []string{"Submit adjustment with any line delta non-zero: status becomes SUBMITTED, version increments, stock posting queued, audit record created."}
	if err := validateRoutingPlanWithContext(skipPC, ChangeInput{}, prematureQueued); !errors.Is(err, ErrAgentPlanIncomplete) {
		t.Fatalf("expected premature 'stock posting queued' on submit to fail, got %v", err)
	}
	prematureMovements := deferredOK
	prematureMovements.ProposedChanges = []string{"If any delta is non-zero, proceed with the existing logic of inserting into `stock_movements`."}
	if err := validateRoutingPlanWithContext(skipPC, ChangeInput{}, prematureMovements); !errors.Is(err, ErrAgentPlanIncomplete) {
		t.Fatalf("expected premature inserting into stock_movements on submit to fail, got %v", err)
	}
	repurposedSummary := FilePlan{
		File:    "backend/test/integration/operations_test.go",
		Layer:   "test",
		Action:  domain.ActionModify,
		Summary: "Update the integration test TestInventoryApprovalRejectsChangedSnapshot in backend/test/integration/operations_test.go to cover the new zero-delta branch (NO_ADJUSTMENT), the opposing delta branch (SUBMITTED), and the stale version rejection (409 Conflict).",
	}
	if err := validateRoutingPlanWithContext(skipPC, ChangeInput{}, repurposedSummary); !errors.Is(err, ErrAgentPlanIncomplete) {
		t.Fatalf("expected summary repurposing TestInventoryApprovalRejectsChangedSnapshot without preservation cue to fail, got %v", err)
	}
	preservedSummary := FilePlan{
		File:             "backend/test/integration/operations_test.go",
		Layer:            "test",
		Action:           domain.ActionModify,
		Summary:          "Keep TestInventoryApprovalRejectsChangedSnapshot unchanged (preserving its original changed-snapshot assertion) and add separate test cases for zero-delta (NO_ADJUSTMENT), opposing non-zero (+2/-2 -> SUBMITTED), and stale version (409 Conflict).",
		Steps:            []string{"Update the existing `TestInventoryApprovalRejectsChangedSnapshot` to retain its assertion on version mismatch returning 409 Conflict, ensuring it does not conflict with the new zero-delta bypass."},
		AcceptanceChecks: []string{"All zero lines -> NO_ADJUSTMENT; opposing +2/-2 lines -> SUBMITTED; stale version -> 409."},
	}
	if err := validateRoutingPlanWithContext(skipPC, ChangeInput{}, preservedSummary); err != nil {
		t.Fatalf("rejected test plan that preserves existing approval test: %v", err)
	}
	invertedOpposing := deferredOK
	invertedOpposing.AcceptanceChecks = []string{"Verify opposing deltas (+2 and -2) trigger NO_ADJUSTMENT status."}
	if err := validateRoutingPlanWithContext(skipPC, ChangeInput{}, invertedOpposing); !errors.Is(err, ErrAgentPlanIncomplete) {
		t.Fatalf("expected +2/-2 -> NO_ADJUSTMENT inversion to fail, got %v", err)
	}
	// Verify dynamic BPMN token extraction works identically on a completely different domain/status set.
	otherDomainPC := &domain.ProcessContext{
		Before: &domain.Graph{Elements: map[string]*domain.Element{"sub": {Name: "Submit to PENDING_REVIEW"}}},
		After: &domain.Graph{
			Elements: map[string]*domain.Element{
				"Zero":  {ID: "Zero", Type: "exclusiveGateway", Name: "All line differences are zero?"},
				"Close": {ID: "AutoSettle", Type: "serviceTask", Name: "Close as AUTO_SETTLED", Documentation: "Skip approval queue; ledger posting is deferred until after approval."},
				"Sub":   {ID: "Sub", Type: "serviceTask", Name: "Route to PENDING_REVIEW"},
			},
			Flows: map[string]*domain.Flow{
				"f1": {SourceRef: "Zero", TargetRef: "Close", Condition: "all line deltas are zero"},
				"f2": {SourceRef: "Zero", TargetRef: "Sub", Condition: "any non-zero delta"},
			},
		},
	}
	otherBad := FilePlan{
		File:             "backend/internal/billing/store.go",
		Layer:            "adapter",
		Action:           domain.ActionModify,
		AcceptanceChecks: []string{"Verify opposing deltas (+2 and -2) trigger AUTO_SETTLED status."},
	}
	if err := validateRoutingPlanWithContext(otherDomainPC, ChangeInput{}, otherBad); !errors.Is(err, ErrAgentPlanIncomplete) || !strings.Contains(err.Error(), "AUTO_SETTLED") || !strings.Contains(err.Error(), "PENDING_REVIEW") {
		t.Fatalf("expected dynamic cross-domain status names in validation error, got: %v", err)
	}
	otherPremature := FilePlan{
		File:             "backend/internal/billing/store.go",
		Layer:            "adapter",
		Action:           domain.ActionModify,
		AcceptanceChecks: []string{"Non-zero delta transitions to PENDING_REVIEW and inserts into ledger_balances."},
	}
	if err := validateRoutingPlanWithContext(otherDomainPC, ChangeInput{}, otherPremature); !errors.Is(err, ErrAgentPlanIncomplete) {
		t.Fatalf("expected premature ledger_balances insert on PENDING_REVIEW submit to fail, got: %v", err)
	}
	// Non-zero-delta Skip scenario (e.g., invoice amount < 1000 skips manager approval) must trigger zero false positives.
	amountSkipPC := &domain.ProcessContext{
		After: &domain.Graph{
			Elements: map[string]*domain.Element{
				"Gate": {ID: "Gate", Type: "exclusiveGateway", Name: "Amount below threshold?"},
				"Skip": {ID: "AutoApprove", Type: "serviceTask", Name: "Auto-approve invoice as AUTO_APPROVED"},
				"Mgr":  {ID: "MgrReview", Type: "userTask", Name: "Manager approval"},
			},
			Flows: map[string]*domain.Flow{
				"f1": {SourceRef: "Gate", TargetRef: "Skip", Condition: "amount < 1000"},
				"f2": {SourceRef: "Gate", TargetRef: "Mgr", Condition: "amount >= 1000"},
			},
		},
	}
	amountSkipPlan := FilePlan{
		File:             "backend/internal/billing/store.go",
		Layer:            "adapter",
		Action:           domain.ActionModify,
		ProposedChanges:  []string{"Calculate the sum of invoice line totals; if amount < 1000 set status to AUTO_APPROVED."},
		AcceptanceChecks: []string{"Invoice below 1000 becomes AUTO_APPROVED; invoice >= 1000 routes to manager approval."},
	}
	if err := validateRoutingPlanWithContext(amountSkipPC, ChangeInput{}, amountSkipPlan); err != nil {
		t.Fatalf("unexpected false positive on non-zero-delta amount Skip scenario: %v", err)
	}
}

func TestCombinedPlanConflictDetectorFlagsSameFileContradictions(t *testing.T) {
	bundle := EvidenceBundle{
		ProcessContext: &domain.ProcessContext{
			After: &domain.Graph{
				Elements: map[string]*domain.Element{
					"ZeroCheck": {ID: "ZeroCheck", Type: "exclusiveGateway", Name: "Semua selisih nol?"},
					"CloseZero": {ID: "CloseZero", Type: "serviceTask", Name: "Tutup NO_ADJUSTMENT"},
					"NormalSub": {ID: "NormalSub", Type: "serviceTask", Name: "Submit SUBMITTED"},
				},
				Flows: map[string]*domain.Flow{
					"f1": {SourceRef: "ZeroCheck", TargetRef: "CloseZero", Condition: "semua selisih nol"},
					"f2": {SourceRef: "ZeroCheck", TargetRef: "NormalSub", Condition: "ada baris dengan selisih nonnol"},
				},
			},
		},
	}
	conflicting := []FilePlan{
		{
			ElementID:       "CheckZeroDelta",
			File:            "backend/store.go",
			Action:          domain.ActionModify,
			ProposedChanges: []string{"When all lines have zero delta, set status to NO_ADJUSTMENT."},
		},
		{
			ElementID:       "SkipOutcomeJoin",
			File:            "backend/store.go",
			Action:          domain.ActionModify,
			ProposedChanges: []string{"When all lines have zero delta, set status to SUBMITTED."},
		},
	}
	issues := combinedPlanConflictIssues(bundle, conflicting)
	if len(issues) == 0 || !strings.Contains(strings.Join(issues, " "), "Cross-proposal conflict on backend/store.go") {
		t.Fatalf("expected cross-proposal conflict issue, got %v", issues)
	}
	consistent := []FilePlan{
		conflicting[0],
		{
			ElementID:       "SkipOutcomeJoin",
			File:            "backend/store.go",
			Action:          domain.ActionModify,
			ProposedChanges: []string{"When all lines have zero delta, preserve NO_ADJUSTMENT through the join."},
		},
	}
	if issues := combinedPlanConflictIssues(bundle, consistent); len(issues) != 0 {
		t.Fatalf("flagged consistent same-file proposals: %v", issues)
	}
}

func TestMisplacedDiagramRecipeIsRejectedWithoutBanningDocumentation(t *testing.T) {
	bad := FilePlan{File: "docs/API-CONTRACT.md", Steps: []string{"Identify the current BPMN process.", "Add a new user task ReasonReview.", "Update the flow connections."}, ProposedChanges: []string{"Insert a new user task in the BPMN process."}}
	if err := validatePlanScope(bad); !errors.Is(err, ErrAgentPlanIncomplete) {
		t.Fatal("uploaded failed plan was accepted", err)
	}
	gateway := FilePlan{File: "docs/api.md", ElementID: "Join", Steps: []string{"Insert the new gateway element into the process definition."}, ProposedChanges: []string{"Add a new BPMN element of type exclusiveGateway."}}
	if !errors.Is(validatePlanScope(gateway), ErrAgentPlanIncomplete) {
		t.Fatal("XML gateway edits accepted for docs")
	}
	md := string(Markdown(ChangePlan{Status: StatusNeedsReview, Plans: []FilePlan{gateway}}))
	if !strings.Contains(md, "still needs supporting evidence") || strings.Contains(md, "Add a new BPMN element") || strings.Contains(md, "NEEDS_REVIEW") {
		t.Fatal(md)
	}
	for _, p := range []FilePlan{
		{File: "service.go", Steps: []string{"Add validation in Submit before state mutation to implement the new BPMN task."}},
		{File: "docs/API-CONTRACT.md", Steps: []string{"Document the reason request field and invalid-input response for submit."}},
		{File: "diagram.bpmn", Steps: bad.Steps},
		{File: "frontend/panel.tsx", Layer: "frontend", Action: domain.ActionModify, ProposedChanges: []string{"Forward the reason input value in the submit request payload."}},
	} {
		if err := validatePlanScope(p); err != nil {
			t.Fatal("valid proposal rejected", err)
		}
	}
	misplacedUI := FilePlan{
		File:   "frontend/features/inventory/inventory-adjustment-panel.tsx",
		Layer:  "frontend",
		Action: domain.ActionModify,
		ProposedChanges: []string{
			"Backend Service: Implement delta calculation logic in the submit handler.",
			"Backend Service: If all deltas are zero, set status to NO_ADJUSTMENT.",
			"Test: Add a unit test for zero delta.",
		},
	}
	if err := validatePlanScope(misplacedUI); !errors.Is(err, ErrAgentPlanIncomplete) {
		t.Fatalf("accepted frontend MODIFY containing only Backend Service/Test steps: %v", err)
	}
}

func TestPlanScopeBlocksClientSnapshotHallucinationAndUnrelatedSubmoduleEntity(t *testing.T) {
	pc := &domain.ProcessContext{
		Before: &domain.Graph{
			ProcessNames: map[string]string{"P": "Submit stock adjustment - Staff"},
			Elements: map[string]*domain.Element{
				"u1": {ID: "u1", Type: "userTask", Name: "Kirim adjustment untuk review"},
			},
		},
		After: &domain.Graph{
			ProcessNames: map[string]string{"P": "Submit stock adjustment - Staff"},
			Elements: map[string]*domain.Element{
				"u1":        {ID: "u1", Type: "userTask", Name: "Kirim adjustment untuk review"},
				"CheckZero": {ID: "CheckZero", Type: "serviceTask", Name: "Periksa apakah semua selisih stok nol"},
			},
		},
	}
	clientDeltaUI := FilePlan{
		File:            "frontend/features/inventory/inventory-adjustment-panel.tsx",
		Layer:           "frontend",
		Action:          domain.ActionModify,
		ProposedChanges: []string{"Modify the create function to include client-side delta calculation logic comparing countedQuantity against a stored snapshot."},
	}
	if err := validatePlanScopeWithContext(pc, Candidate{File: clientDeltaUI.File}, clientDeltaUI); !errors.Is(err, ErrAgentPlanIncomplete) {
		t.Fatalf("accepted client-side snapshot calculation when no userTask changed: %v", err)
	}
	// Forwarding reason on Gateway scenario must still pass!
	gatewayReasonUI := FilePlan{
		File:            "frontend/features/inventory/inventory-adjustment-panel.tsx",
		Layer:           "frontend",
		Action:          domain.ActionModify,
		ProposedChanges: []string{"Update action('submit') to forward the reason state value in the request payload."},
	}
	if err := validatePlanScopeWithContext(pc, Candidate{File: gatewayReasonUI.File}, gatewayReasonUI); err != nil {
		t.Fatalf("rejected valid payload-forwarding UI edit: %v", err)
	}
	// Sibling sales submodule editing TransitionOrder on a stock adjustment process must be rejected.
	wrongSubmodule := FilePlan{
		File:            "backend/internal/sales/adapter/postgres/store.go",
		Layer:           "persistence",
		Action:          domain.ActionModify,
		Symbol:          "Store.TransitionOrder",
		ProposedChanges: []string{"Inside TransitionOrder, set status to NO_ADJUSTMENT when deltas are zero."},
	}
	if err := validatePlanScopeWithContext(pc, Candidate{File: wrongSubmodule.File}, wrongSubmodule); !errors.Is(err, ErrAgentPlanIncomplete) {
		t.Fatalf("accepted unrelated sales/store.go TransitionOrder edit: %v", err)
	}
	// Matching inventory submodule editing TransitionAdjustment must pass!
	rightSubmodule := FilePlan{
		File:            "backend/internal/inventory/adapter/postgres/store.go",
		Layer:           "persistence",
		Action:          domain.ActionModify,
		Symbol:          "Store.TransitionAdjustment",
		ProposedChanges: []string{"Inside TransitionAdjustment, set status to NO_ADJUSTMENT when all line deltas are zero."},
	}
	if err := validatePlanScopeWithContext(pc, Candidate{File: rightSubmodule.File}, rightSubmodule); err != nil {
		t.Fatalf("rejected valid inventory/store.go TransitionAdjustment edit: %v", err)
	}
}

func TestDocumentationDoesNotHideUnresolvedImplementationCoverage(t *testing.T) {
	b := agentBundle()
	p := FilePlan{ChangeType: b.Changes[0].ChangeType, ElementID: b.Changes[0].ElementID, File: "docs/api.md", Layer: "documentation"}
	issues := implementationCoverageIssues(b, []FilePlan{p})
	if len(issues) != 1 || !strings.Contains(issues[0], "implementation candidates remain unresolved") {
		t.Fatal(issues)
	}
	code := p
	code.File = "service.go"
	code.Layer = "application"
	if issues := implementationCoverageIssues(b, []FilePlan{p, code}); len(issues) != 0 {
		t.Fatal(issues)
	}
	if issues := implementationCoverageIssues(b, nil); len(issues) != 0 {
		t.Fatal("invented a documentation proposal", issues)
	}
}

func TestInvalidOutputFeedbackIsRoleSpecific(t *testing.T) {
	analyst := modelOutputFeedback(Analyst, errors.New("unknown target index"))
	if !strings.Contains(analyst, "unknown target index") || !strings.Contains(analyst, "verification.decisions") || strings.Contains(analyst, "current_behavior") {
		t.Fatal(analyst)
	}
	if !strings.Contains(modelOutputFeedback(Planner, errors.New("invalid structured action")), "plan.plans") {
		t.Fatal("planner format absent")
	}
	if !strings.Contains(modelOutputFeedback(Reviewer, errors.New("empty response")), "review action") {
		t.Fatal("review format absent")
	}
}

func TestAgentDoesNotExportPlanForWeakTarget(t *testing.T) {
	b := agentBundle()
	decision := CandidateDecision{ChangeType: b.Changes[0].ChangeType, ElementID: b.Changes[0].ElementID, File: "store.go", Symbol: "Submit", Verdict: VerdictWeak, EvidenceIDs: []string{"proof"}}
	turn := AgentTurn{Role: Planner, Bundle: b}
	reply, _ := goodReply(turn)
	s := AgentSession{Role: Planner, Bundle: b, Decisions: []CandidateDecision{decision}}
	if err := (AgentEngine{}).apply(context.Background(), &s, reply, b); !errors.Is(err, ErrAgentPlanIncomplete) {
		t.Fatal("uncertain plan exported", err)
	}
}
