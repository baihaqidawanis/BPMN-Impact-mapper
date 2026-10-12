// Package reviewpack prepares method-blind reviewer inputs from completed
// canonical reports. It does not infer expected impacts or call the analyzer.
package reviewpack

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/evaluation"
)

const Version = "1"

type Input struct {
	Report domain.Report
	SHA256 string
}

type PredictionReference struct {
	Method     domain.AnalysisMethod `json:"method"`
	AnalysisID string                `json:"analysis_id"`
	SHA256     string                `json:"sha256"`
}

type ElementSnapshot struct {
	ID    string `json:"id"`
	Name  string `json:"name,omitempty"`
	Type  string `json:"type"`
	Actor string `json:"actor,omitempty"`
}

type Case struct {
	ChangeType domain.ChangeType     `json:"change_type"`
	ElementID  string                `json:"element_id"`
	OldElement *ElementSnapshot      `json:"old_element,omitempty"`
	NewElement *ElementSnapshot      `json:"new_element,omitempty"`
	Reason     string                `json:"reason,omitempty"`
	Context    *domain.ChangeContext `json:"context,omitempty"`
}

type Manifest struct {
	Version          string                `json:"version"`
	PackID           string                `json:"pack_id"`
	InputFingerprint string                `json:"input_fingerprint"`
	Repository       domain.Repository     `json:"repository"`
	CaseDigest       string                `json:"case_digest"`
	Predictions      []PredictionReference `json:"predictions"`
	Cases            []Case                `json:"cases"`
}

type AnnotationTemplate struct {
	Version    string            `json:"version"`
	ReviewerID string            `json:"reviewer_id"`
	Cases      []evaluation.Case `json:"cases"`
}

type Pack struct {
	Manifest  Manifest
	ReviewerA AnnotationTemplate
	ReviewerB AnnotationTemplate
}

func Build(inputs []Input, reviewerA, reviewerB string) (Pack, error) {
	if len(inputs) == 0 || len(inputs) > 16 {
		return Pack{}, errors.New("review pack requires between 1 and 16 prediction reports")
	}
	reviewerA, reviewerB = strings.TrimSpace(reviewerA), strings.TrimSpace(reviewerB)
	if reviewerA == "" || reviewerB == "" {
		return Pack{}, errors.New("two reviewer identifiers are required")
	}
	if reviewerA == reviewerB {
		return Pack{}, errors.New("reviewer identifiers must be distinct")
	}

	ordered := append([]Input(nil), inputs...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Report.Method < ordered[j].Report.Method })
	base := ordered[0].Report
	if err := validateReport(base, ordered[0].SHA256); err != nil {
		return Pack{}, fmt.Errorf("report 1: %w", err)
	}
	cases, err := casesFrom(base.Changes)
	if err != nil {
		return Pack{}, err
	}
	caseBytes, err := json.Marshal(cases)
	if err != nil {
		return Pack{}, err
	}
	caseDigest := hash(caseBytes)

	references := make([]PredictionReference, 0, len(ordered))
	seenMethods := map[domain.AnalysisMethod]bool{}
	seenAnalyses := map[string]bool{}
	for index, input := range ordered {
		if err := validateReport(input.Report, input.SHA256); err != nil {
			return Pack{}, fmt.Errorf("report %d: %w", index+1, err)
		}
		if seenMethods[input.Report.Method] {
			return Pack{}, fmt.Errorf("duplicate prediction method %q", input.Report.Method)
		}
		seenMethods[input.Report.Method] = true
		if seenAnalyses[input.Report.AnalysisID] {
			return Pack{}, fmt.Errorf("duplicate prediction analysis_id %q", input.Report.AnalysisID)
		}
		seenAnalyses[input.Report.AnalysisID] = true
		if input.Report.InputFingerprint != base.InputFingerprint {
			return Pack{}, errors.New("prediction reports have different input fingerprints")
		}
		if input.Report.Repository != base.Repository {
			return Pack{}, errors.New("prediction reports have different repository metadata")
		}
		candidateCases, err := casesFrom(input.Report.Changes)
		if err != nil {
			return Pack{}, fmt.Errorf("report %d: %w", index+1, err)
		}
		candidateBytes, err := json.Marshal(candidateCases)
		if err != nil {
			return Pack{}, err
		}
		if !bytes.Equal(caseBytes, candidateBytes) {
			return Pack{}, errors.New("prediction reports describe different BPMN change sets")
		}
		references = append(references, PredictionReference{Method: input.Report.Method, AnalysisID: input.Report.AnalysisID, SHA256: strings.ToLower(input.SHA256)})
	}

	manifest := Manifest{Version: Version, InputFingerprint: base.InputFingerprint, Repository: base.Repository, CaseDigest: caseDigest, Predictions: references, Cases: cases}
	manifest.PackID, err = packID(manifest)
	if err != nil {
		return Pack{}, err
	}
	templateCases := make([]evaluation.Case, 0, len(cases))
	for _, item := range cases {
		// The deliberately invalid action makes an untouched template fail closed
		// in reconciliation. Reviewers replace the sentinel with findings, or
		// remove it and leave [] to record an explicit negative judgment.
		templateCases = append(templateCases, evaluation.Case{ChangeType: item.ChangeType, ElementID: item.ElementID, Expected: []evaluation.ExpectedImpact{{File: "REVIEW_REQUIRED", Action: ""}}})
	}
	return Pack{
		Manifest:  manifest,
		ReviewerA: AnnotationTemplate{Version: Version, ReviewerID: reviewerA, Cases: templateCases},
		ReviewerB: AnnotationTemplate{Version: Version, ReviewerID: reviewerB, Cases: templateCases},
	}, nil
}

// ValidateManifest detects inconsistent coordinator metadata before it is used
// to bind completed reviewer annotations. It is an integrity check, not a
// digital signature against a malicious coordinator.
func ValidateManifest(manifest Manifest) error {
	if manifest.Version != Version || manifest.PackID == "" || manifest.InputFingerprint == "" {
		return errors.New("review manifest requires version, pack_id, and input_fingerprint")
	}
	if strings.TrimSpace(manifest.Repository.Name) == "" || strings.TrimSpace(manifest.Repository.Revision) == "" {
		return errors.New("review manifest requires repository name and revision")
	}
	if len(manifest.Predictions) == 0 || len(manifest.Predictions) > 16 {
		return errors.New("review manifest requires between 1 and 16 predictions")
	}
	if len(manifest.Cases) == 0 {
		return errors.New("review manifest requires at least one case")
	}
	seenCases := map[string]bool{}
	previousCase := ""
	for index, item := range manifest.Cases {
		if item.ChangeType == "" || strings.TrimSpace(item.ElementID) == "" {
			return fmt.Errorf("review manifest case %d is invalid", index)
		}
		key := string(item.ChangeType) + "\x00" + item.ElementID
		if seenCases[key] {
			return fmt.Errorf("review manifest duplicates case %s/%s", item.ChangeType, item.ElementID)
		}
		if index > 0 && key <= previousCase {
			return errors.New("review manifest cases are not canonically ordered")
		}
		seenCases[key] = true
		previousCase = key
	}
	caseBytes, err := json.Marshal(manifest.Cases)
	if err != nil {
		return err
	}
	if hash(caseBytes) != manifest.CaseDigest {
		return errors.New("review manifest case digest mismatch")
	}
	seenMethods := map[domain.AnalysisMethod]bool{}
	seenAnalyses := map[string]bool{}
	previousMethod := domain.AnalysisMethod("")
	for index, prediction := range manifest.Predictions {
		if !prediction.Method.Valid() || prediction.AnalysisID == "" {
			return fmt.Errorf("review manifest prediction %d is invalid", index)
		}
		if seenMethods[prediction.Method] || seenAnalyses[prediction.AnalysisID] {
			return errors.New("review manifest contains duplicate method or analysis_id")
		}
		if index > 0 && prediction.Method <= previousMethod {
			return errors.New("review manifest predictions are not canonically ordered")
		}
		if len(prediction.SHA256) != 64 {
			return fmt.Errorf("review manifest prediction %d has invalid SHA-256", index)
		}
		if _, err := hex.DecodeString(prediction.SHA256); err != nil || prediction.SHA256 != strings.ToLower(prediction.SHA256) {
			return fmt.Errorf("review manifest prediction %d has invalid SHA-256", index)
		}
		seenMethods[prediction.Method], seenAnalyses[prediction.AnalysisID] = true, true
		previousMethod = prediction.Method
	}
	expectedID, err := packID(manifest)
	if err != nil {
		return err
	}
	if expectedID != manifest.PackID {
		return errors.New("review manifest pack_id mismatch")
	}
	return nil
}

// ValidateCompletedCases requires exact manifest coverage and rejects the
// fail-closed sentinel through the ordinary ground-truth validator.
func ValidateCompletedCases(manifest Manifest, cases []evaluation.Case) error {
	if err := ValidateManifest(manifest); err != nil {
		return err
	}
	truth := evaluation.GroundTruth{Version: Version, Cases: cases}
	if err := truth.Validate(); err != nil {
		return err
	}
	expected := map[string]bool{}
	for _, item := range manifest.Cases {
		expected[string(item.ChangeType)+"\x00"+item.ElementID] = true
	}
	if len(cases) != len(expected) {
		return errors.New("review annotation does not cover every manifest case exactly once")
	}
	for _, item := range cases {
		key := string(item.ChangeType) + "\x00" + item.ElementID
		if !expected[key] {
			return fmt.Errorf("review annotation contains case outside manifest: %s/%s", item.ChangeType, item.ElementID)
		}
	}
	return nil
}

func packID(manifest Manifest) (string, error) {
	seed := struct {
		Version          string
		InputFingerprint string
		Repository       domain.Repository
		CaseDigest       string
		Predictions      []PredictionReference
	}{manifest.Version, manifest.InputFingerprint, manifest.Repository, manifest.CaseDigest, manifest.Predictions}
	seedBytes, err := json.Marshal(seed)
	if err != nil {
		return "", err
	}
	return hash(seedBytes)[:32], nil
}

func validateReport(report domain.Report, digest string) error {
	if report.AnalysisID == "" || report.InputFingerprint == "" {
		return errors.New("analysis_id and input_fingerprint are required")
	}
	if !report.Method.Valid() {
		return fmt.Errorf("invalid method %q", report.Method)
	}
	if strings.TrimSpace(report.Repository.Name) == "" || strings.TrimSpace(report.Repository.Revision) == "" {
		return errors.New("repository name and revision are required")
	}
	if len(digest) != 64 {
		return errors.New("prediction SHA-256 must contain 64 hexadecimal characters")
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return errors.New("prediction SHA-256 must contain 64 hexadecimal characters")
	}
	return nil
}

func casesFrom(changes []domain.Change) ([]Case, error) {
	if len(changes) == 0 {
		return nil, errors.New("prediction report contains no BPMN changes to review")
	}
	result := make([]Case, 0, len(changes))
	seen := map[string]bool{}
	for index, change := range changes {
		elementID := ""
		if change.NewElement != nil {
			elementID = change.NewElement.ID
		} else if change.OldElement != nil {
			elementID = change.OldElement.ID
		}
		if change.ChangeType == "" || strings.TrimSpace(elementID) == "" {
			return nil, fmt.Errorf("change %d requires change_type and element ID", index)
		}
		key := string(change.ChangeType) + "\x00" + elementID
		if seen[key] {
			return nil, fmt.Errorf("duplicate change case %s/%s", change.ChangeType, elementID)
		}
		seen[key] = true
		result = append(result, Case{ChangeType: change.ChangeType, ElementID: elementID, OldElement: snapshot(change.OldElement), NewElement: snapshot(change.NewElement), Reason: change.Reason, Context: change.Context})
	}
	sort.Slice(result, func(i, j int) bool {
		return string(result[i].ChangeType)+"\x00"+result[i].ElementID < string(result[j].ChangeType)+"\x00"+result[j].ElementID
	})
	return result, nil
}

func snapshot(element *domain.Element) *ElementSnapshot {
	if element == nil {
		return nil
	}
	return &ElementSnapshot{ID: element.ID, Name: element.Name, Type: element.Type, Actor: element.Actor}
}

func JSON(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func Guide(manifest Manifest) []byte {
	var output bytes.Buffer
	output.WriteString("# Blind BPMN Change Review\n\n")
	output.WriteString("Inspect the frozen repository revision independently. Do not use mapper rankings, scores, or candidate lists while annotating. For every case, replace the REVIEW_REQUIRED sentinel with all expected repository-relative files and required actions. Remove the sentinel and use an empty expected list only for an explicit negative judgment. Untouched templates are intentionally rejected by reconciliation.\n\n")
	fmt.Fprintf(&output, "- Repository: %s\n- Revision: %s\n- Cases: %d\n\n", inline(manifest.Repository.Name), inline(manifest.Repository.Revision), len(manifest.Cases))
	for index, item := range manifest.Cases {
		fmt.Fprintf(&output, "## %d. %s — `%s`\n\n", index+1, item.ChangeType, inline(item.ElementID))
		if item.OldElement != nil {
			fmt.Fprintf(&output, "- Before: %s\n", describe(item.OldElement))
		}
		if item.NewElement != nil {
			fmt.Fprintf(&output, "- After: %s\n", describe(item.NewElement))
		}
		if item.Reason != "" {
			fmt.Fprintf(&output, "- Change reason: %s\n", inline(item.Reason))
		}
		if item.Context != nil {
			fmt.Fprintf(&output, "- BPMN context: %s; predecessors: %s; successors: %s\n", inline(string(item.Context.Pattern)), contextNames(item.Context.Predecessors), contextNames(item.Context.Successors))
		}
		output.WriteByte('\n')
	}
	output.WriteString("Return only the completed reviewer annotation JSON. The coordinator keeps snapshot provenance separately and reconciles both reviews after completion.\n")
	return output.Bytes()
}

func describe(element *ElementSnapshot) string {
	parts := []string{"`" + inline(element.ID) + "`", inline(element.Type)}
	if element.Name != "" {
		parts = append(parts, "\""+inline(element.Name)+"\"")
	}
	if element.Actor != "" {
		parts = append(parts, "actor "+inline(element.Actor))
	}
	return strings.Join(parts, "; ")
}

func contextNames(values []domain.ElementContext) string {
	if len(values) == 0 {
		return "none"
	}
	items := make([]string, 0, len(values))
	for _, value := range values {
		label := inline(value.ID)
		if value.Name != "" {
			label += " (" + inline(value.Name) + ")"
		}
		items = append(items, label)
	}
	return strings.Join(items, ", ")
}

func inline(value string) string {
	value = strings.NewReplacer("`", "'", "\r", " ", "\n", " ", "|", "/").Replace(value)
	return strings.Join(strings.Fields(value), " ")
}

func hash(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}
