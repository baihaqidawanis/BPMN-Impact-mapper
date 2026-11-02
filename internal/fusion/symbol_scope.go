package fusion

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/textutil"
)

// resolveSymbolScope does not rerank files or alter RRF scores. A retrieved
// passage may identify a file without establishing which function must change.
func resolveSymbolScope(change *domain.Change, artifacts []domain.Artifact) {
	byFile := map[string][]domain.Artifact{}
	for _, artifact := range artifacts {
		if artifact.Symbol.Kind == "function" || artifact.Symbol.Kind == "method" || artifact.Symbol.Kind == "type" {
			byFile[artifact.File] = append(byFile[artifact.File], artifact)
		}
	}
	ids := changedIDs(*change)
	labelTokens := changedLabelTokens(*change)
	for index := range change.Impacts {
		impact := &change.Impacts[index]
		if impact.Symbol == "" {
			continue
		}
		fileSymbols := byFile[impact.File]
		var selected *domain.Artifact
		for i := range fileSymbols {
			if symbolName(fileSymbols[i].Symbol) == impact.Symbol {
				selected = &fileSymbols[i]
				break
			}
		}
		// A Go package name is not an affected function. Unknown adapter
		// symbols remain untouched until that adapter has a scope contract.
		if selected == nil && !isPackageArtifact(artifacts, impact.File, impact.Symbol) {
			continue
		}
		if selected != nil && (uniqueDirectID(*selected, fileSymbols, ids) || uniqueLabelMatch(*selected, fileSymbols, labelTokens)) {
			continue
		}
		original := impact.Symbol
		impact.Symbol = ""
		for evidenceIndex := range impact.Evidence {
			if impact.Evidence[evidenceIndex].Type == "RRF_FUSION" && impact.Evidence[evidenceIndex].Location == impact.File+" / "+original {
				impact.Evidence[evidenceIndex].Location = impact.File
			}
		}
		impact.Evidence = uniqueEvidence(append(impact.Evidence, domain.Evidence{
			Type: "SYMBOL_SCOPE_UNRESOLVED", Location: impact.File,
			Excerpt: "File retrieved; source symbol " + original + " is not uniquely supported by the changed BPMN ID or at least two label words. Review the relevant function manually.",
		}))
	}
}

func isPackageArtifact(artifacts []domain.Artifact, file, name string) bool {
	for _, artifact := range artifacts {
		if artifact.File == file && artifact.Symbol.Kind == "package" && symbolName(artifact.Symbol) == name {
			return true
		}
	}
	return false
}

func changedIDs(change domain.Change) []string {
	ids := []string{}
	for _, element := range []*domain.Element{change.OldElement, change.NewElement} {
		if element != nil && element.ID != "" {
			ids = append(ids, element.ID)
		}
	}
	return textutil.UniqueSorted(ids)
}

func changedLabelTokens(change domain.Change) map[string]bool {
	result := map[string]bool{}
	for _, element := range []*domain.Element{change.OldElement, change.NewElement} {
		if element == nil {
			continue
		}
		for _, token := range textutil.Tokenize(element.Name) {
			if len([]rune(token)) >= 4 {
				result[token] = true
			}
		}
	}
	return result
}

func uniqueDirectID(selected domain.Artifact, symbols []domain.Artifact, ids []string) bool {
	for _, id := range ids {
		if !containsDelimitedID(selected.Content, id) {
			continue
		}
		matches := 0
		for _, symbol := range symbols {
			if containsDelimitedID(symbol.Content, id) {
				matches++
			}
		}
		if matches == 1 {
			return true
		}
	}
	return false
}

func uniqueLabelMatch(selected domain.Artifact, symbols []domain.Artifact, labelTokens map[string]bool) bool {
	selectedMatches := symbolLabelMatches(selected.Symbol, labelTokens)
	if selectedMatches < 2 {
		return false
	}
	for _, symbol := range symbols {
		if symbol.ID != selected.ID && symbolLabelMatches(symbol.Symbol, labelTokens) >= selectedMatches {
			return false
		}
	}
	return true
}

func symbolLabelMatches(symbol domain.Symbol, labelTokens map[string]bool) int {
	seen := map[string]bool{}
	for _, token := range textutil.Tokenize(symbol.Name) {
		if labelTokens[token] {
			seen[token] = true
		}
	}
	return len(seen)
}

func symbolName(symbol domain.Symbol) string {
	if symbol.Receiver != "" {
		return symbol.Receiver + "." + symbol.Name
	}
	return symbol.Name
}

func containsDelimitedID(content, id string) bool {
	if id == "" {
		return false
	}
	for start := 0; start < len(content); {
		index := strings.Index(content[start:], id)
		if index < 0 {
			return false
		}
		index += start
		end := index + len(id)
		leftOK := index == 0
		if !leftOK {
			previous, _ := utf8.DecodeLastRuneInString(content[:index])
			leftOK = !identifierRune(previous)
		}
		rightOK := end == len(content)
		if !rightOK {
			next, _ := utf8.DecodeRuneInString(content[end:])
			rightOK = !identifierRune(next)
		}
		if leftOK && rightOK {
			return true
		}
		start = index + 1
	}
	return false
}

func identifierRune(value rune) bool {
	return unicode.IsLetter(value) || unicode.IsDigit(value) || value == '_' || value == '-'
}
