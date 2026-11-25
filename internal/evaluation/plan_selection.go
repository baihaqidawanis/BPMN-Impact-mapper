package evaluation

import (
	"errors"
	"sort"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

// PlanEntry represents one planned file target from an agent change plan.
// Keeping this value type in package evaluation preserves decoupling from
// package planning while enforcing the exact thesis file-selection protocol.
type PlanEntry struct {
	ChangeType domain.ChangeType     `json:"change_type"`
	ElementID  string                `json:"element_id"`
	File       string                `json:"file"`
	Action     domain.RequiredAction `json:"action"`
}

// PlanSelectionCase reports per-change file-selection accuracy for a single
// frozen ground-truth scope, separating concrete edits (ADD/MODIFY/REMOVE)
// from excluded non-edit assessments (NO_ACTION/VERIFY/UNRESOLVED).
type PlanSelectionCase struct {
	ChangeType      domain.ChangeType `json:"change_type"`
	ElementID       string            `json:"element_id"`
	ExpectedFiles   []string          `json:"expected_files"`
	ProposedEdits   []string          `json:"proposed_edits"`
	ExcludedNonEdit []string          `json:"excluded_non_edit,omitempty"`
	TruePositives   int               `json:"true_positives"`
	FalsePositives  int               `json:"false_positives"`
	FalseNegatives  int               `json:"false_negatives"`
	Precision       float64           `json:"precision"`
	Recall          float64           `json:"recall"`
	F1              float64           `json:"f1"`
	MandatoryFound  bool              `json:"mandatory_found"`
	NoChangeControl bool              `json:"no_change_control,omitempty"`
	FalseEdit       bool              `json:"false_edit,omitempty"`
}

// PlanSelectionReport summarizes agent file-selection performance across a
// frozen ground-truth reference following docs/METHODOLOGY.md:
//   - deduplicate normalized existing file paths within each case scope;
//   - exclude NO_ACTION, VERIFY, and UNRESOLVED from proposed edits;
//   - score positive cases with equal weight for macro precision/recall/F1
//     (zero when a positive case has no proposed edits);
//   - report no-change controls separately by whether they trigger a false edit.
type PlanSelectionReport struct {
	PositiveCases       int                 `json:"positive_cases"`
	MacroPrecision      float64             `json:"macro_precision"`
	MacroRecall         float64             `json:"macro_recall"`
	MacroF1             float64             `json:"macro_f1"`
	AllMandatoryFound   bool                `json:"all_mandatory_found"`
	NoChangeControls    int                 `json:"no_change_controls"`
	FalseEditControls   int                 `json:"false_edit_controls"`
	ExcludedNonEditRows int                 `json:"excluded_non_edit_rows"`
	Cases               []PlanSelectionCase `json:"cases"`
}

func isProposedEditAction(action domain.RequiredAction) bool {
	switch action {
	case domain.ActionAdd, domain.ActionModify, domain.ActionRemove:
		return true
	default:
		return false
	}
}

// EvaluatePlanSelection evaluates agent file-selection proposals against a
// frozen GroundTruth scope without mutating inputs.
func EvaluatePlanSelection(entries []PlanEntry, truth GroundTruth) (PlanSelectionReport, error) {
	if err := truth.Validate(); err != nil {
		return PlanSelectionReport{}, err
	}
	if len(truth.Cases) == 0 {
		return PlanSelectionReport{}, errors.New("ground truth requires at least one case")
	}
	editsByCase := map[string]map[string]bool{}
	excludedByCase := map[string]map[string]bool{}
	excludedRows := 0
	for _, entry := range entries {
		file := normalizePath(entry.File)
		if file == "" {
			continue
		}
		key := caseKey(entry.ChangeType, entry.ElementID)
		if isProposedEditAction(entry.Action) {
			if editsByCase[key] == nil {
				editsByCase[key] = map[string]bool{}
			}
			editsByCase[key][file] = true
		} else {
			excludedRows++
			if excludedByCase[key] == nil {
				excludedByCase[key] = map[string]bool{}
			}
			excludedByCase[key][file] = true
		}
	}

	sortedCases := append([]Case(nil), truth.Cases...)
	sort.Slice(sortedCases, func(i, j int) bool {
		return caseKey(sortedCases[i].ChangeType, sortedCases[i].ElementID) < caseKey(sortedCases[j].ChangeType, sortedCases[j].ElementID)
	})

	report := PlanSelectionReport{
		AllMandatoryFound:   true,
		ExcludedNonEditRows: excludedRows,
		Cases:               make([]PlanSelectionCase, 0, len(sortedCases)),
	}
	var sumP, sumR float64
	for _, truthCase := range sortedCases {
		key := caseKey(truthCase.ChangeType, truthCase.ElementID)
		expected := expectedMap(truthCase.Expected)
		edits := editsByCase[key]
		excluded := excludedByCase[key]
		var excludedFiles []string
		for f := range excluded {
			if !edits[f] {
				excludedFiles = append(excludedFiles, f)
			}
		}
		sort.Strings(excludedFiles)

		proposed := fileSetKeys(edits)
		item := PlanSelectionCase{
			ChangeType:      truthCase.ChangeType,
			ElementID:       truthCase.ElementID,
			ExpectedFiles:   sortedKeys(expected),
			ProposedEdits:   proposed,
			ExcludedNonEdit: excludedFiles,
		}
		if len(expected) == 0 {
			item.NoChangeControl = true
			item.FalsePositives = len(proposed)
			item.FalseEdit = len(proposed) > 0
			item.MandatoryFound = !item.FalseEdit
			report.NoChangeControls++
			if item.FalseEdit {
				report.FalseEditControls++
			}
			report.Cases = append(report.Cases, item)
			continue
		}

		tp, fp, fn := 0, 0, 0
		for _, f := range proposed {
			if _, ok := expected[f]; ok {
				tp++
			} else {
				fp++
			}
		}
		for f := range expected {
			if !edits[f] {
				fn++
			}
		}
		item.TruePositives = tp
		item.FalsePositives = fp
		item.FalseNegatives = fn
		item.Precision = ratio(tp, tp+fp)
		item.Recall = ratio(tp, tp+fn)
		if item.Precision+item.Recall > 0 {
			item.F1 = round(2 * item.Precision * item.Recall / (item.Precision + item.Recall))
		}
		item.MandatoryFound = fn == 0
		if !item.MandatoryFound {
			report.AllMandatoryFound = false
		}
		report.PositiveCases++
		sumP += item.Precision
		sumR += item.Recall
		report.Cases = append(report.Cases, item)
	}
	if report.PositiveCases > 0 {
		report.MacroPrecision = round(sumP / float64(report.PositiveCases))
		report.MacroRecall = round(sumR / float64(report.PositiveCases))
		if report.MacroPrecision+report.MacroRecall > 0 {
			report.MacroF1 = round(2 * report.MacroPrecision * report.MacroRecall / (report.MacroPrecision + report.MacroRecall))
		}
	}
	return report, nil
}
