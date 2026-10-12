package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/evaluation"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/report"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/review"
)

type reviewerPair struct {
	A review.Annotation `json:"reviewer_a"`
	B review.Annotation `json:"reviewer_b"`
}

type evaluationRequest struct {
	ReviewerA   review.Annotation `json:"reviewer_a"`
	ReviewerB   review.Annotation `json:"reviewer_b"`
	Resolutions []evaluation.Case `json:"resolutions"`
	AnalysisIDs []string          `json:"analysis_ids"`
	Fingerprint string            `json:"input_fingerprint"`
	Confirmed   bool              `json:"snapshot_confirmed"`
}

func decodeEvaluationInput(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeAPIError(w, http.StatusBadRequest, "INVALID_REVIEW", "Invalid or oversized review JSON (maximum 2 MiB).")
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeAPIError(w, http.StatusBadRequest, "INVALID_REVIEW", "Exactly one JSON object is required.")
		return false
	}
	return true
}

func (h Handler) reconcileReviews(w http.ResponseWriter, r *http.Request) {
	var input reviewerPair
	if !decodeEvaluationInput(w, r, &input) {
		return
	}
	result, err := review.Reconcile(input.A, input.B)
	if err != nil {
		writeAPIError(w, 422, "INVALID_REVIEW", "Use version-1 annotations with distinct reviewer IDs, valid actions and relative files.")
		return
	}
	writeJSON(w, 200, result)
}

// Evaluates completed immutable predictions only. No analyzer or repository is accessed.
func (h Handler) evaluateReports(w http.ResponseWriter, r *http.Request) {
	var input evaluationRequest
	if !decodeEvaluationInput(w, r, &input) {
		return
	}
	if h.Store == nil {
		h.internalError(w, errors.New("report store unavailable"))
		return
	}
	if !input.Confirmed || input.Fingerprint == "" || len(input.AnalysisIDs) < 1 || len(input.AnalysisIDs) > 4 {
		writeAPIError(w, 422, "INVALID_EVALUATION", "Confirm the review snapshot and select one to four stored reports.")
		return
	}
	truth, err := review.Adjudicate(input.ReviewerA, input.ReviewerB, input.Resolutions)
	if err != nil {
		writeAPIError(w, 422, "UNRESOLVED_REVIEW", "Every disagreement needs exactly one explicit valid resolution.")
		return
	}
	predictions := []domain.Report{}
	methods := map[domain.AnalysisMethod]bool{}
	for _, id := range input.AnalysisIDs {
		if r.Context().Err() != nil {
			return
		}
		prediction, err := h.Store.Get(r.Context(), id)
		if err != nil {
			writeAPIError(w, 404, "REPORT_UNAVAILABLE", "A selected stored report is unavailable.")
			return
		}
		if prediction.InputFingerprint != input.Fingerprint || methods[prediction.Method] {
			writeAPIError(w, 422, "INPUT_MISMATCH", "Reports must share the selected fingerprint and have unique methods.")
			return
		}
		if err := evaluation.ValidateCoverage(prediction, truth); err != nil {
			writeAPIError(w, 422, "COVERAGE_MISMATCH", "Ground truth must cover every selected BPMN change exactly once, with no extra cases.")
			return
		}
		methods[prediction.Method] = true
		predictions = append(predictions, prediction)
	}
	sort.Slice(predictions, func(i, j int) bool { return predictions[i].Method < predictions[j].Method })
	truthBytes, _ := json.MarshalIndent(truth, "", "  ")
	results := []evaluation.Report{}
	markdown := "# Method evaluation\n\nFile-level evaluation. False discovery proportion = FP/(TP+FP); true-negative-based FPR is unavailable. Latency is not measured here.\n\n"
	for _, prediction := range predictions {
		encoded, err := report.JSON(prediction)
		if err != nil {
			h.internalError(w, err)
			return
		}
		result, err := evaluation.Evaluate(prediction, encoded, truth, truthBytes)
		if err != nil {
			writeAPIError(w, 422, "INVALID_EVALUATION", "Ground truth does not match the selected report.")
			return
		}
		results = append(results, result)
		markdown += string(evaluation.Markdown(result)) + "\n"
	}
	writeJSON(w, 200, struct {
		Fingerprint string                 `json:"input_fingerprint"`
		Truth       evaluation.GroundTruth `json:"ground_truth"`
		Results     []evaluation.Report    `json:"results"`
		Markdown    string                 `json:"markdown"`
		ReviewerA   review.Annotation      `json:"reviewer_a"`
		ReviewerB   review.Annotation      `json:"reviewer_b"`
		Resolutions []evaluation.Case      `json:"resolutions"`
	}{input.Fingerprint, truth, results, markdown, input.ReviewerA, input.ReviewerB, input.Resolutions})
}
