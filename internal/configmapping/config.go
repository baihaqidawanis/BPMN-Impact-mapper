// Package configmapping implements the explicit configuration-driven mapping
// adapter. It depends inward on domain contracts and the mapping strategy port.
package configmapping

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
)

type Config struct {
	Version string `json:"version"`
	Rules   []Rule `json:"rules"`
}

type Rule struct {
	ChangeType domain.ChangeType     `json:"change_type,omitempty"`
	ElementID  string                `json:"element_id,omitempty"`
	File       string                `json:"file"`
	Symbol     string                `json:"symbol,omitempty"`
	Module     string                `json:"module,omitempty"`
	Layer      string                `json:"layer,omitempty"`
	Action     domain.RequiredAction `json:"required_action"`
	Rationale  string                `json:"rationale"`
}

type Mapper struct{ Config Config }

func Load(file string) (Mapper, error) {
	info, err := os.Lstat(file)
	if err != nil {
		return Mapper{}, fmt.Errorf("mapping configuration: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return Mapper{}, errors.New("mapping configuration must be a regular non-symlink file")
	}
	if info.Size() > 1<<20 {
		return Mapper{}, errors.New("mapping configuration exceeds 1 MiB")
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return Mapper{}, fmt.Errorf("read mapping configuration: %w", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return Mapper{}, fmt.Errorf("decode mapping configuration: %w", err)
	}
	if err := config.Validate(); err != nil {
		return Mapper{}, err
	}
	sort.Slice(config.Rules, func(i, j int) bool {
		left := fmt.Sprintf("%s|%s|%s|%s", config.Rules[i].ChangeType, config.Rules[i].ElementID, config.Rules[i].File, config.Rules[i].Symbol)
		right := fmt.Sprintf("%s|%s|%s|%s", config.Rules[j].ChangeType, config.Rules[j].ElementID, config.Rules[j].File, config.Rules[j].Symbol)
		return left < right
	})
	return Mapper{Config: config}, nil
}

func (config Config) Validate() error {
	if config.Version != "1" {
		return fmt.Errorf("unsupported mapping configuration version %q", config.Version)
	}
	seen := map[string]bool{}
	for index, rule := range config.Rules {
		if rule.ChangeType == "" && strings.TrimSpace(rule.ElementID) == "" {
			return fmt.Errorf("rule %d requires change_type or element_id selector", index)
		}
		if rule.ChangeType != "" && !validChangeType(rule.ChangeType) {
			return fmt.Errorf("rule %d has unsupported change_type %q", index, rule.ChangeType)
		}
		if err := validatePath(rule.File); err != nil {
			return fmt.Errorf("rule %d: %w", index, err)
		}
		if !validAction(rule.Action) {
			return fmt.Errorf("rule %d has unsupported action %q", index, rule.Action)
		}
		if strings.TrimSpace(rule.Rationale) == "" {
			return fmt.Errorf("rule %d requires rationale", index)
		}
		key := fmt.Sprintf("%s|%s|%s|%s", rule.ChangeType, rule.ElementID, rule.File, rule.Symbol)
		if seen[key] {
			return fmt.Errorf("duplicate mapping rule %q", key)
		}
		seen[key] = true
	}
	return nil
}

func (Mapper) Method() domain.AnalysisMethod { return domain.MethodConfiguration }

func (mapper Mapper) Attach(changes []domain.Change, artifacts []domain.Artifact, _ mapping.Options) ([]domain.Change, []domain.Warning) {
	byFile := map[string][]domain.Artifact{}
	for _, artifact := range artifacts {
		byFile[artifact.File] = append(byFile[artifact.File], artifact)
	}
	warnings := []domain.Warning{}
	for changeIndex := range changes {
		change := &changes[changeIndex]
		change.Impacts = []domain.Impact{}
		if change.ChangeType == domain.UnresolvedMatch {
			continue
		}
		for _, rule := range mapper.Config.Rules {
			if !matches(rule, *change) {
				continue
			}
			artifact, ok := target(byFile[rule.File], rule.Symbol)
			if !ok {
				warnings = append(warnings, domain.Warning{Code: "CONFIG_TARGET_MISSING", Message: fmt.Sprintf("mapping target %s#%s is absent from the allowed index", rule.File, rule.Symbol)})
				continue
			}
			module, layer := artifact.Module, artifact.Layer
			if rule.Module != "" {
				module = rule.Module
			}
			if rule.Layer != "" {
				layer = rule.Layer
			}
			change.Impacts = append(change.Impacts, domain.Impact{
				Module: module, Layer: layer, File: rule.File, Symbol: rule.Symbol,
				RequiredAction: rule.Action,
				Evidence:       []domain.Evidence{{Type: "CONFIG_MAPPING", Location: rule.File, Excerpt: bounded(rule.Rationale, 240)}},
				ScoreBreakdown: domain.ScoreBreakdown{Total: 1}, Confidence: 1,
			})
		}
		sort.Slice(change.Impacts, func(i, j int) bool {
			if change.Impacts[i].File != change.Impacts[j].File {
				return change.Impacts[i].File < change.Impacts[j].File
			}
			return change.Impacts[i].Symbol < change.Impacts[j].Symbol
		})
		if len(change.Impacts) == 0 {
			change.UnresolvedImpact = append(change.UnresolvedImpact, "configuration has no valid indexed mapping for this change")
		}
	}
	sort.Slice(warnings, func(i, j int) bool { return warnings[i].Message < warnings[j].Message })
	return changes, warnings
}

func matches(rule Rule, change domain.Change) bool {
	if rule.ChangeType != "" && rule.ChangeType != change.ChangeType {
		return false
	}
	if rule.ElementID == "" {
		return true
	}
	id := ""
	if change.NewElement != nil {
		id = change.NewElement.ID
	} else if change.OldElement != nil {
		id = change.OldElement.ID
	}
	return rule.ElementID == id
}

func target(artifacts []domain.Artifact, symbol string) (domain.Artifact, bool) {
	if len(artifacts) == 0 {
		return domain.Artifact{}, false
	}
	if symbol == "" {
		return artifacts[0], true
	}
	for _, artifact := range artifacts {
		display := artifact.Symbol.Name
		if artifact.Symbol.Receiver != "" {
			display = artifact.Symbol.Receiver + "." + artifact.Symbol.Name
		}
		if display == symbol {
			return artifact, true
		}
	}
	return domain.Artifact{}, false
}

func validatePath(value string) error {
	if value == "" || strings.Contains(value, "\\") || path.IsAbs(value) {
		return errors.New("file must be a non-empty slash-normalized relative path")
	}
	clean := path.Clean(value)
	if clean != value || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return errors.New("file must be a clean contained relative path")
	}
	return nil
}

func validAction(action domain.RequiredAction) bool {
	switch action {
	case domain.ActionAdd, domain.ActionModify, domain.ActionRemove, domain.ActionVerify, domain.ActionNoAction, domain.ActionUnresolved:
		return true
	default:
		return false
	}
}

func validChangeType(change domain.ChangeType) bool {
	switch change {
	case domain.AddTask, domain.RemoveTask, domain.ModifyTaskLabel, domain.ModifyActorRole,
		domain.AddGateway, domain.RemoveGateway, domain.ModifyGatewayCondition, domain.ModifySequenceFlow,
		domain.AddDataRequirement, domain.RemoveDataRequirement, domain.AddState,
		domain.ModifyTerminalOutcome, domain.UnresolvedMatch:
		return true
	default:
		return false
	}
}

func bounded(value string, limit int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit-1]) + "…"
}
