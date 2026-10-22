package relationprobe

import "github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"

type Move struct {
	File         string `json:"file"`
	OriginalRank int    `json:"original_rank"`
	ProposedRank int    `json:"proposed_rank"`
	Hops         int    `json:"hops"`
}

type Proposal struct {
	ChangeType domain.ChangeType `json:"change_type"`
	ElementID  string            `json:"element_id"`
	BeforeTop5 []string          `json:"before_top_5"`
	AfterTop5  []string          `json:"after_top_5"`
	Moves      []Move            `json:"moves"`
}

// Propose is a score-free ablation: move existing, source-linked files directly
// after their cited seed in path order. It never creates a file candidate,
// changes confidence/actions, or mutates the canonical report.
func Propose(report domain.Report, findings []Finding) []Proposal {
	result := []Proposal{}
	for _, change := range report.Changes {
		files := make([]string, 0, len(change.Impacts))
		original := map[string]int{}
		for _, impact := range change.Impacts {
			if impact.File != "" && original[impact.File] == 0 {
				files = append(files, impact.File)
				original[impact.File] = len(files)
			}
		}
		if len(files) == 0 {
			continue
		}
		proposal := Proposal{ChangeType: change.ChangeType, ElementID: elementID(change), BeforeTop5: topFive(files), Moves: []Move{}}
		moved := map[string]bool{}
		for _, link := range findings {
			if link.ChangeType != proposal.ChangeType || link.ElementID != proposal.ElementID || original[link.TargetFile] == 0 || moved[link.TargetFile] {
				continue
			}
			seedPosition := position(files, link.SeedFile)
			targetPosition := position(files, link.TargetFile)
			if seedPosition < 0 || targetPosition <= seedPosition {
				continue
			}
			anchor := link.SeedFile
			if len(link.Steps) == 3 {
				for _, earlier := range findings {
					if earlier.ChangeType == link.ChangeType && earlier.ElementID == link.ElementID && earlier.SeedFile == link.SeedFile && len(earlier.Steps) == 2 && moved[earlier.TargetFile] {
						anchor = earlier.TargetFile
						break
					}
				}
			}
			files = append(files[:targetPosition], files[targetPosition+1:]...)
			insertAt := position(files, anchor) + 1
			files = append(files, "")
			copy(files[insertAt+1:], files[insertAt:])
			files[insertAt] = link.TargetFile
			moved[link.TargetFile] = true
			proposal.Moves = append(proposal.Moves, Move{File: link.TargetFile, OriginalRank: original[link.TargetFile], Hops: len(link.Steps) - 1})
		}
		if len(proposal.Moves) == 0 {
			continue
		}
		for i := range proposal.Moves {
			proposal.Moves[i].ProposedRank = position(files, proposal.Moves[i].File) + 1
		}
		proposal.AfterTop5 = topFive(files)
		result = append(result, proposal)
	}
	return result
}

func elementID(change domain.Change) string {
	if change.NewElement != nil {
		return change.NewElement.ID
	}
	if change.OldElement != nil {
		return change.OldElement.ID
	}
	return ""
}

func position(values []string, target string) int {
	for i, value := range values {
		if value == target {
			return i
		}
	}
	return -1
}

func topFive(values []string) []string {
	end := len(values)
	if end > 5 {
		end = 5
	}
	return append([]string(nil), values[:end]...)
}
