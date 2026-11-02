package planning

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

// Byte ranges are half-open and overlap. Content is the full range, not a
// retrieved excerpt. No source file is ever executed.
type RepositoryChunk struct {
	HybridLeads []string `json:"hybrid_leads,omitempty"`
	HybridRank  int      `json:"hybrid_rank,omitempty"`
	ID          string   `json:"id"`
	File        string   `json:"file"`
	Module      string   `json:"module"`
	Layer       string   `json:"layer"`
	StartByte   int      `json:"start_byte"`
	EndByte     int      `json:"end_byte"`
	FileBytes   int      `json:"file_bytes"`
	LastInFile  bool     `json:"last_in_file"`
	Content     string   `json:"content"`
}

type FullReviewProgress struct {
	SourceFingerprint string `json:"source_fingerprint,omitempty"`
	Phase             string `json:"phase"`
	FilesTotal        int    `json:"files_total"`
	FilesReviewed     int    `json:"files_reviewed"`
	ChunksTotal       int    `json:"chunks_total"`
	ChunksReviewed    int    `json:"chunks_reviewed"`
	FailedChunks      int    `json:"failed_chunks"`
	RelevantFiles     int    `json:"relevant_files"`
	WeakFiles         int    `json:"weak_files"`
	BatchesTotal      int    `json:"batches_total"`
	BatchesReviewed   int    `json:"batches_reviewed"`
	CurrentFile       string `json:"current_file,omitempty"`
}

type RepositorySource interface {
	SourceTools
	RepositoryChunks() []RepositoryChunk
}

// FullRepositoryEngine checkpoints each valid chunk decision separately. It
// then plans all non-rejected files in batches, never truncating them to top-k.
type FullRepositoryEngine struct {
	Model  AgentModel
	Source RepositorySource
	Store  AgentSessionStore
}

func (e FullRepositoryEngine) Run(ctx context.Context, id string, original EvidenceBundle) (AgentSession, error) {
	var s AgentSession
	if e.Model == nil || e.Source == nil || e.Store == nil || !validAgentID(id) || len(id) > 40 || len(original.Changes) != 1 {
		return s, errors.New("full review dependencies and one selected change required")
	}
	if err := validateBundle(original); err != nil {
		return s, err
	}
	chunks := append([]RepositoryChunk(nil), e.Source.RepositoryChunks()...)
	if len(chunks) == 0 {
		return s, errors.New("no eligible source files to review")
	}
	leadRank := map[string]int{}
	leads := []string{}
	for _, candidate := range original.Changes[0].Candidates {
		if _, ok := leadRank[candidate.File]; !ok {
			leads = append(leads, candidate.File)
			leadRank[candidate.File] = len(leads)
		}
	}
	for i := range chunks {
		chunks[i].HybridLeads = leads
		chunks[i].HybridRank = leadRank[chunks[i].File]
	}
	sort.SliceStable(chunks, func(i, j int) bool {
		left, right := chunks[i].HybridRank, chunks[j].HybridRank
		if left == 0 {
			left = len(leads) + 1
		}
		if right == 0 {
			right = len(leads) + 1
		}
		if left != right {
			return left < right
		}
		if chunks[i].File != chunks[j].File {
			return chunks[i].File < chunks[j].File
		}
		return chunks[i].StartByte < chunks[j].StartByte
	})
	files := map[string]bool{}
	ids := map[string]bool{}
	for i, c := range chunks {
		if !agentRelativeFile(c.File) || len(c.Content) > 4096 || c.ID == "" || ids[c.ID] || c.StartByte < 0 || c.EndByte-c.StartByte != len(c.Content) || c.EndByte > c.FileBytes || c.FileBytes < 0 {
			return s, errors.New("invalid source chunk")
		}
		ids[c.ID] = true
		if i == 0 || chunks[i-1].File != c.File {
			if c.StartByte != 0 {
				return s, errors.New("source manifest omits file beginning")
			}
		} else {
			prior := chunks[i-1]
			if prior.LastInFile || prior.FileBytes != c.FileBytes || c.StartByte > prior.EndByte || c.EndByte <= prior.EndByte {
				return s, errors.New("source manifest contains a gap or invalid overlap")
			}
		}
		last := i == len(chunks)-1 || chunks[i+1].File != c.File
		if c.LastInFile != last || (last && c.EndByte != c.FileBytes) {
			return s, errors.New("source manifest omits file end")
		}
		files[c.File] = true
	}
	identity := e.Model.Identity()
	if identity.Provider == "" || identity.Model == "" || identity.PromptVersion == "" || e.Source.Identity() == "" {
		return s, errors.New("full review identities required")
	}
	binding := agentDigest([]any{"full-repository-v1", original.ContextID, e.Source.Identity(), identity})
	loaded, err := e.Store.Load(ctx, id)
	if errors.Is(err, ErrAgentSessionMissing) {
		s = AgentSession{Version: "1", ID: id, Binding: binding, Identity: identity, Bundle: original, Role: Analyst, Status: "RUNNING", FullReview: &FullReviewProgress{SourceFingerprint: e.Source.Identity(), Phase: "scanning", FilesTotal: len(files), ChunksTotal: len(chunks)}}
	} else if err != nil {
		return s, err
	} else {
		s = loaded
		if s.Binding != binding || s.FullReview == nil {
			return AgentSession{}, errors.New("full review source/model/change binding changed; start a new session")
		}
		if s.Status == "COMPLETE" || s.Status == "NEEDS_REVIEW" {
			return s, nil
		}
	}
	if s.FullReview.ChunksReviewed < 0 || s.FullReview.ChunksReviewed > len(chunks) {
		return s, errors.New("invalid full review cursor")
	}
	save := func() error { return e.Store.Save(ctx, s) }
	pause := func(message string, cause error) (AgentSession, error) {
		s.Status = "PAUSED"
		s.Feedback = []string{message}
		checkpoint, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		if err := e.Store.Save(checkpoint, s); err != nil {
			return s, err
		}
		return s, cause
	}
	// Rebuild the immutable scan aggregate from durable receipts. A crash after a
	// receipt write but before cursor update does not require another model call.
	candidates := map[string]Candidate{}
	decisions := map[string]CandidateDecision{}
	scanTurns := 0
	for i, c := range chunks {
		if err := ctx.Err(); err != nil {
			return pause("Full review interrupted; resume from the saved chunk.", err)
		}
		childID := fmt.Sprintf("%s-scan-%d", id, i)
		receipt, loadErr := e.Store.Load(ctx, childID)
		chunkBinding := agentDigest([]any{binding, c.ID})
		if loadErr != nil && !errors.Is(loadErr, ErrAgentSessionMissing) {
			return pause("Chunk checkpoint unavailable.", loadErr)
		}
		if loadErr == nil && receipt.Binding != chunkBinding {
			return pause("Chunk source binding mismatch.", errors.New("chunk binding mismatch"))
		}
		if loadErr != nil {
			change := original.Changes[0]
			evidence := EvidenceItem{ID: c.ID, Type: "FULL_SOURCE_CHUNK", Location: fmt.Sprintf("%s@bytes:%d-%d", c.File, c.StartByte, c.EndByte), Excerpt: truncateChunkExcerpt(c.Content)}
			change.Candidates = []Candidate{{File: c.File, Module: c.Module, Layer: c.Layer, SuggestedAction: domain.ActionVerify, Origin: "full-repository-review", Evidence: []EvidenceItem{evidence}}}
			bundle := original
			bundle.Changes = []ChangeInput{change}
			bundle.ContextID, _ = ComputeContextID(bundle)
			receipt = AgentSession{Version: "1", ID: childID, Binding: chunkBinding, Identity: identity, Bundle: bundle, Role: Analyst, Status: "RUNNING", ReviewedChunk: &c}
		}
		if receipt.Status != "COMPLETE" {
			s.Role = Analyst
			s.Status = "RUNNING"
			s.FullReview.Phase = "scanning"
			s.FullReview.CurrentFile = c.File
			if err := save(); err != nil {
				return s, err
			}
			if receipt.Turns >= 3 {
				s.FullReview.FailedChunks = 1
				s.Status = "NEEDS_REVIEW"
				s.Feedback = []string{"A source chunk failed three attempts; coverage is incomplete."}
				_ = save()
				return s, nil
			}
			receipt.Turns++
			receipt.Status = "RUNNING"
			if err := e.Store.Save(ctx, receipt); err != nil {
				return s, err
			}
			callCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
			reply, callErr := e.Model.Turn(callCtx, AgentTurn{Role: Analyst, Bundle: receipt.Bundle, RemainingTools: 0, RepositoryChunk: &c})
			cancel()
			s.Turns++
			if callErr != nil {
				return pause("Source chunk inference interrupted. Completed chunks are preserved.", callErr)
			}
			if reply.Verification == nil || reply.Tool != nil || reply.Plan != nil || reply.Review != nil || reply.NeedEvidence != nil {
				return pause("Invalid chunk classification; resume to retry.", ErrAgentModelOutput)
			}
			if err := validateDecisions(receipt.Bundle.Changes[0], reply.Verification.Decisions); err != nil {
				return pause("Chunk evidence failed validation; resume to retry.", err)
			}
			receipt.Decisions = reply.Verification.Decisions
			receipt.Status = "COMPLETE"
			if err := e.Store.Save(ctx, receipt); err != nil {
				return s, err
			}
		}
		if len(receipt.Decisions) != 1 {
			return pause("Invalid chunk receipt.", errors.New("missing chunk decision"))
		}
		scanTurns += receipt.Turns
		d := receipt.Decisions[0]
		candidate := receipt.Bundle.Changes[0].Candidates[0]
		prior, exists := decisions[c.File]
		if !exists || verdictPriority(d.Verdict) > verdictPriority(prior.Verdict) {
			decisions[c.File] = d
			candidates[c.File] = candidate
		} else if d.Verdict == prior.Verdict && len(candidates[c.File].Evidence) < 8 {
			merged := candidates[c.File]
			merged.Evidence = append(merged.Evidence, candidate.Evidence...)
			candidates[c.File] = merged
		}
		if i >= s.FullReview.ChunksReviewed {
			s.FullReview.ChunksReviewed = i + 1
			if c.LastInFile {
				s.FullReview.FilesReviewed++
			}
			if err := save(); err != nil {
				return s, err
			}
		}
	}
	names := make([]string, 0, len(candidates))
	for name := range candidates {
		names = append(names, name)
	}
	sort.Strings(names)
	finalBundle := original
	change := original.Changes[0]
	change.Candidates = nil
	s.Decisions = nil
	s.Draft = nil
	s.Feedback = nil
	var pending []string
	s.FullReview.RelevantFiles = 0
	s.FullReview.WeakFiles = 0
	for _, name := range names {
		candidate := candidates[name]
		d := decisions[name]
		// Every retained evidence item is allowed; retain the strongest chunk's
		// rationale while making its actual citations available to planning.
		change.Candidates = append(change.Candidates, candidate)
		s.Decisions = append(s.Decisions, d)
		if d.Verdict != VerdictReject {
			pending = append(pending, name)
		}
		if d.Verdict == VerdictRelevant {
			s.FullReview.RelevantFiles++
		}
		if d.Verdict == VerdictWeak {
			s.FullReview.WeakFiles++
		}
	}
	finalBundle.Changes = []ChangeInput{change}
	finalBundle.ContextID, _ = ComputeContextID(finalBundle)
	s.Bundle = finalBundle
	s.Turns = scanTurns
	s.Tools = 0
	s.Revisions = 0
	s.FullReview.Phase = "planning"
	s.FullReview.BatchesTotal = (len(pending) + 2) / 3
	issues := []string{}
	for start := 0; start < len(pending); start += 3 {
		batchIndex := start / 3
		batchBundle := original
		batchChange := original.Changes[0]
		batchChange.Candidates = nil
		var verified []CandidateDecision
		for _, name := range pending[start:min(start+3, len(pending))] {
			batchChange.Candidates = append(batchChange.Candidates, candidates[name])
			verified = append(verified, decisions[name])
		}
		batchBundle.Changes = []ChangeInput{batchChange}
		batchBundle.ContextID, _ = ComputeContextID(batchBundle)
		s.Role = Planner
		s.Status = "RUNNING"
		s.FullReview.CurrentFile = pending[start]
		if err := save(); err != nil {
			return s, err
		}
		engine := AgentEngine{Model: e.Model, Tools: e.Source, Store: e.Store, InitialDecisions: verified, Budget: AgentBudget{MaxTurns: 18, MaxTools: 8, MaxRevisions: 2, MaxDiscoveries: 0}}
		batch, err := engine.Run(ctx, fmt.Sprintf("%s-plan-%d", id, batchIndex), batchBundle)
		s.Turns += batch.Turns
		s.Tools += batch.Tools
		s.Revisions += batch.Revisions
		if err != nil {
			return pause("File planning interrupted; the complete source scan is saved.", err)
		}
		if _, err := batch.ChangePlan(); err != nil {
			s.Status = "NEEDS_REVIEW"
			s.Feedback = []string{"Source scan completed but a planning batch has no valid plan."}
			_ = save()
			return s, nil
		}
		s.Draft = append(s.Draft, batch.Draft...)
		for _, candidate := range batch.Bundle.Changes[0].Candidates {
			for i, existing := range s.Bundle.Changes[0].Candidates {
				if existing.File == candidate.File {
					s.Bundle.Changes[0].Candidates[i] = candidate
				}
			}
		}
		s.Bundle.ContextID, _ = ComputeContextID(s.Bundle)
		for _, d := range batch.Decisions {
			for i, existing := range s.Decisions {
				if existing.File == d.File {
					s.Decisions[i] = d
				}
			}
		}
		if batch.Status != "COMPLETE" {
			issues = append(issues, batch.Feedback...)
			if len(batch.Feedback) == 0 {
				issues = append(issues, "A planning batch still needs review.")
			}
		}
		s.FullReview.BatchesReviewed = batchIndex + 1
		if err := save(); err != nil {
			return s, err
		}
	}
	s.Status = "COMPLETE"
	s.Role = Reviewer
	s.Feedback = cleanStrings(issues)
	s.FullReview.Phase = "finished"
	s.FullReview.CurrentFile = ""
	s.FullReview.RelevantFiles = 0
	s.FullReview.WeakFiles = 0
	for _, d := range s.Decisions {
		if d.Verdict == VerdictRelevant {
			s.FullReview.RelevantFiles++
		}
		if d.Verdict == VerdictWeak {
			s.FullReview.WeakFiles++
		}
	}
	if len(issues) > 0 || agentIncomplete(s.Bundle) {
		s.Status = "NEEDS_REVIEW"
	}
	if err := save(); err != nil {
		return s, err
	}
	return s, nil
}

func verdictPriority(v Verdict) int {
	switch v {
	case VerdictRelevant:
		return 3
	case VerdictWeak:
		return 2
	default:
		return 1
	}
}
func truncateChunkExcerpt(s string) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) > 240 {
		r = r[:240]
	}
	return string(r)
}
