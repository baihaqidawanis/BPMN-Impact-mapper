package api

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/report"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/reviewpack"
)

type reviewPackRequest struct {
	AnalysisIDs []string `json:"analysis_ids"`
	Fingerprint string   `json:"input_fingerprint"`
	ReviewerA   string   `json:"reviewer_a"`
	ReviewerB   string   `json:"reviewer_b"`
}

type reviewPackResponse struct {
	Manifest  reviewpack.Manifest           `json:"manifest"`
	ReviewerA reviewpack.AnnotationTemplate `json:"reviewer_a"`
	ReviewerB reviewpack.AnnotationTemplate `json:"reviewer_b"`
	Guide     string                        `json:"guide_markdown"`
}

// prepareReviewPack loads immutable completed predictions only. It does not
// invoke the analyzer and accepts no expected impacts or ground truth.
func (h Handler) prepareReviewPack(w http.ResponseWriter, r *http.Request) {
	var input reviewPackRequest
	if !decodeEvaluationInput(w, r, &input) {
		return
	}
	if h.Store == nil {
		h.internalError(w, errors.New("report store unavailable"))
		return
	}
	input.ReviewerA, input.ReviewerB = strings.TrimSpace(input.ReviewerA), strings.TrimSpace(input.ReviewerB)
	if input.Fingerprint == "" || len(input.AnalysisIDs) < 1 || len(input.AnalysisIDs) > 4 || input.ReviewerA == "" || input.ReviewerB == "" || input.ReviewerA == input.ReviewerB {
		writeAPIError(w, 422, "INVALID_REVIEW_PACK", "Select one to four same-snapshot reports and two distinct reviewer IDs.")
		return
	}
	inputs := make([]reviewpack.Input, 0, len(input.AnalysisIDs))
	for _, id := range input.AnalysisIDs {
		if r.Context().Err() != nil {
			return
		}
		prediction, err := h.Store.Get(r.Context(), id)
		if err != nil {
			writeAPIError(w, 404, "REPORT_UNAVAILABLE", "A selected stored report is unavailable.")
			return
		}
		if prediction.InputFingerprint != input.Fingerprint {
			writeAPIError(w, 422, "INPUT_MISMATCH", "Reports must match the selected input fingerprint.")
			return
		}
		encoded, err := report.JSON(prediction)
		if err != nil {
			h.internalError(w, err)
			return
		}
		digest := sha256.Sum256(encoded)
		inputs = append(inputs, reviewpack.Input{Report: prediction, SHA256: hex.EncodeToString(digest[:])})
	}
	pack, err := reviewpack.Build(inputs, input.ReviewerA, input.ReviewerB)
	if err != nil {
		writeAPIError(w, 422, "INVALID_REVIEW_PACK", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, reviewPackResponse{Manifest: pack.Manifest, ReviewerA: pack.ReviewerA, ReviewerB: pack.ReviewerB, Guide: string(reviewpack.Guide(pack.Manifest))})
}
