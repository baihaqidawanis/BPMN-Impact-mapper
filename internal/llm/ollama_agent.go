package llm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

type OllamaAgentConfig struct {
	BaseURL     string
	Model       string
	ContextSize int
	MaxOutput   int
	Client      *http.Client
}

// OllamaAgent uses Ollama JSON-schema output for one action per turn. Tool
// execution is performed only by AgentEngine, never by the model endpoint.
type OllamaAgent struct {
	base, model            string
	contextSize, maxOutput int
	client                 *http.Client
	mu                     sync.RWMutex
	digest, version        string
}

func NewOllamaAgent(c OllamaAgentConfig) (*OllamaAgent, error) {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, errors.New("invalid Ollama URL")
	}
	ip := net.ParseIP(u.Hostname())
	if ip == nil || !ip.IsLoopback() || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Opaque != "" || u.Fragment != "" || u.Path != "" {
		return nil, errors.New("Ollama requires a numeric loopback HTTP URL without credentials/path/query")
	}
	if strings.TrimSpace(c.Model) == "" || strings.Contains(strings.ToLower(c.Model), ":cloud") {
		return nil, errors.New("explicit local model required; cloud models are forbidden")
	}
	if c.ContextSize == 0 {
		c.ContextSize = 16384
	}
	if c.MaxOutput == 0 {
		c.MaxOutput = 4096
	}
	if c.ContextSize < 4096 || c.ContextSize > 32768 || c.MaxOutput < 32 || c.MaxOutput > 8192 || c.MaxOutput >= c.ContextSize {
		return nil, errors.New("Ollama context/output budget invalid")
	}
	client := http.Client{Timeout: 85 * time.Second, Transport: &http.Transport{Proxy: nil}}
	if c.Client != nil {
		client = *c.Client
		if client.Timeout <= 0 {
			client.Timeout = 85 * time.Second
		}
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &OllamaAgent{base: c.BaseURL, model: c.Model, contextSize: c.ContextSize, maxOutput: c.MaxOutput, client: &client}, nil
}

func (m *OllamaAgent) request(ctx context.Context, method, path string, payload any, out any) error {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}
	if len(body) > 64<<10 {
		return errors.New("Ollama request exceeds bounded context")
	}
	r, err := http.NewRequestWithContext(ctx, method, m.base+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json")
	resp, err := m.client.Do(r)
	if err != nil {
		return errors.New("Ollama connection interrupted; check VPN/tunnel")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama HTTP %d; response omitted", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil || len(data) > 1<<20 {
		return errors.New("Ollama response unavailable or oversized")
	}
	if err := json.Unmarshal(data, out); err != nil {
		return errors.New("Ollama returned invalid JSON")
	}
	return nil
}

func (m *OllamaAgent) localDigest(ctx context.Context) (string, error) {
	var tags struct {
		Models []struct {
			Name         string   `json:"name"`
			Digest       string   `json:"digest"`
			RemoteHost   string   `json:"remote_host"`
			RemoteModel  string   `json:"remote_model"`
			Capabilities []string `json:"capabilities"`
		} `json:"models"`
	}
	if err := m.request(ctx, http.MethodGet, "/api/tags", nil, &tags); err != nil {
		return "", err
	}
	for _, tag := range tags.Models {
		if tag.Name == m.model {
			if tag.RemoteHost != "" || tag.RemoteModel != "" {
				return "", errors.New("remote/cloud Ollama model refused")
			}
			decoded, err := hex.DecodeString(tag.Digest)
			if err != nil || len(decoded) != 32 {
				return "", errors.New("model digest missing/invalid")
			}
			hasTools := false
			for _, c := range tag.Capabilities {
				if c == "tools" {
					hasTools = true
				}
			}
			if !hasTools {
				return "", errors.New("selected model does not advertise tools capability")
			}
			return tag.Digest, nil
		}
	}
	return "", errors.New("selected Ollama model not installed; no automatic pull")
}

// Prepare inspects metadata only; it does not load a model or send source.
func (m *OllamaAgent) Prepare(ctx context.Context) error {
	digest, err := m.localDigest(ctx)
	if err != nil {
		return err
	}
	var runtime struct {
		Version string `json:"version"`
	}
	if err := m.request(ctx, http.MethodGet, "/api/version", nil, &runtime); err != nil {
		return err
	}
	if runtime.Version == "" {
		return errors.New("Ollama runtime version missing")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.digest != "" && (m.digest != digest || m.version != runtime.Version) {
		return errors.New("Ollama identity changed; create a new adapter/session")
	}
	m.digest = digest
	m.version = runtime.Version
	return nil
}

func (m *OllamaAgent) Identity() planning.ProviderIdentity {
	m.mu.RLock()
	defer m.mu.RUnlock()
	contracts := []any{}
	for _, role := range []planning.AgentRole{planning.Analyst, planning.Planner, planning.Reviewer} {
		schema, _ := agentActionSchema(role)
		contracts = append(contracts, ollamaAgentInstruction(role), schema)
	}
	data, _ := json.Marshal([]any{"ollama-agent-json-v3", contracts, fullChunkInstructions, m.digest, m.version, m.contextSize, m.maxOutput, false, 0})
	sum := sha256.Sum256(data)
	return planning.ProviderIdentity{Provider: "ollama-local", Model: m.model, PromptVersion: "ollama-agent-json-v3-" + hex.EncodeToString(sum[:])[:24]}
}

func (m *OllamaAgent) Turn(ctx context.Context, t planning.AgentTurn) (planning.AgentReply, error) {
	var out planning.AgentReply
	m.mu.RLock()
	pinned := m.digest
	m.mu.RUnlock()
	if pinned == "" {
		return out, errors.New("prepare Ollama metadata before agent execution")
	}
	digest, err := m.localDigest(ctx)
	if err != nil {
		return out, err
	}
	if digest != pinned {
		return out, errors.New("Ollama model digest drift")
	}
	inputTurn := t
	switch t.Role {
	case planning.Analyst:
		inputTurn.Draft = nil
		inputTurn.Decisions = nil
		inputTurn.Observations = compactAgentObservations(t.Observations)
	case planning.Planner:
		inputTurn.Observations = nil
	case planning.Reviewer:
		inputTurn.Decisions = nil
		inputTurn.Observations = nil
	}
	input, err := json.Marshal(inputTurn)
	if err != nil {
		return out, err
	}
	if len(input) > 24<<10 {
		return out, fmt.Errorf("%w: use a smaller evidence bundle", planning.ErrAgentContextLimit)
	}
	schema, err := agentActionSchema(t.Role)
	if err != nil {
		return out, err
	}
	if t.Role == planning.Analyst && t.RemainingTools <= 0 {
		schema = map[string]any{"type": "object", "properties": map[string]any{"verification": schemaType(reflect.TypeOf(planning.VerificationResponse{}))}, "required": []string{"verification"}, "additionalProperties": false}
	}
	if t.Role == planning.Analyst && t.RemainingTools > 0 {
		if file, changeIndex := unreadAgentSource(t); file != "" {
			// A search hit cannot establish what a handler does. Require one
			// real source window per candidate file before final decisions,
			// subject to the existing tool/turn budgets.
			tool := map[string]any{"type": "object", "properties": map[string]any{
				"name":         map[string]any{"type": "string", "enum": []string{"inspect_source"}},
				"change_index": map[string]any{"type": "integer", "enum": []int{changeIndex}},
				"file":         map[string]any{"type": "string", "enum": []string{file}},
				"query":        map[string]any{"type": "string", "enum": []string{agentInspectionQuery(t, changeIndex)}},
			}, "required": []string{"name", "change_index", "file", "query"}, "additionalProperties": false}
			schema = map[string]any{"type": "object", "properties": map[string]any{"tool": tool}, "required": []string{"tool"}, "additionalProperties": false}
		}
	}
	if t.Role == planning.Analyst && t.RemainingTools > 0 && !agentSearchedRepository(t) {
		tool := map[string]any{"type": "object", "properties": map[string]any{
			"name":         map[string]any{"type": "string", "enum": []string{"search_source"}},
			"change_index": map[string]any{"type": "integer", "minimum": 0, "maximum": max(0, len(t.Bundle.Changes)-1)},
			"query":        map[string]any{"type": "string", "minLength": 1, "maxLength": 200},
		}, "required": []string{"name", "change_index", "query"}, "additionalProperties": false}
		schema = map[string]any{"type": "object", "properties": map[string]any{"tool": tool}, "required": []string{"tool"}, "additionalProperties": false}
	}
	// Cardinality follows the current bundle, including tool discoveries. Schema
	// constraints complement (never replace) engine identity/evidence validation.
	if t.Role == planning.Analyst {
		count := 0
		for _, change := range t.Bundle.Changes {
			count += len(change.Candidates)
		}
		constrainActionArray(schema, "verification", "decisions", count)
	}
	if t.Role == planning.Planner {
		count := 0
		for _, decision := range t.Decisions {
			if decision.Verdict != planning.VerdictReject {
				count++
			}
		}
		constrainActionArray(schema, "plan", "plans", count)
	}
	ids := []string{}
	seenIDs := map[string]bool{}
	for _, change := range t.Bundle.Changes {
		for _, c := range change.Candidates {
			for _, e := range c.Evidence {
				if !seenIDs[e.ID] {
					ids = append(ids, e.ID)
					seenIDs[e.ID] = true
				}
			}
		}
	}
	if t.Role == planning.Analyst {
		constrainActionEvidence(schema, "verification", "decisions", ids)
	}
	if t.Role == planning.Planner {
		constrainActionEvidence(schema, "plan", "plans", ids)
	}
	if t.Role == planning.Analyst {
		bindAgentTargets(schema, "verification", "decisions", t, false)
	}
	if t.Role == planning.Planner {
		bindAgentTargets(schema, "plan", "plans", t, true)
	}
	instruction := ollamaAgentInstruction(t.Role)
	if t.RepositoryChunk != nil {
		if t.Role != planning.Analyst || len(t.RepositoryChunk.Content) > 4096 {
			return out, errors.New("invalid full review chunk turn")
		}
		instruction += fullChunkInstructions
	}
	schemaBytes, _ := json.Marshal(schema)
	// Byte count is a conservative tokenizer-independent upper bound, with a
	// template reserve. Reject instead of relying on server-side input truncation.
	if len(input)+len(instruction)+len(schemaBytes) > m.contextSize-m.maxOutput-512 {
		return out, fmt.Errorf("%w: export fewer candidates/evidence or increase configured context", planning.ErrAgentContextLimit)
	}
	payload := map[string]any{"model": m.model, "stream": false, "think": false, "keep_alive": "2m", "format": schema, "options": map[string]any{"num_ctx": m.contextSize, "num_predict": m.maxOutput, "temperature": 0}, "messages": []map[string]string{{"role": "system", "content": instruction}, {"role": "user", "content": string(input)}}}
	var response struct {
		Model   string `json:"model"`
		Done    bool   `json:"done"`
		Reason  string `json:"done_reason"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := m.request(ctx, http.MethodPost, "/api/chat", payload, &response); err != nil {
		return out, err
	}
	if !response.Done || response.Reason == "length" || response.Model != m.model || response.Message.Role != "assistant" || strings.TrimSpace(response.Message.Content) == "" {
		return out, fmt.Errorf("%w: completion incomplete or identity invalid", planning.ErrAgentModelOutput)
	}
	d := json.NewDecoder(strings.NewReader(response.Message.Content))
	d.DisallowUnknownFields()
	if err := d.Decode(&out); err != nil {
		return out, fmt.Errorf("%w: invalid structured action", planning.ErrAgentModelOutput)
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return out, fmt.Errorf("%w: trailing data", planning.ErrAgentModelOutput)
	}
	n := 0
	for _, present := range []bool{out.Tool != nil, out.Verification != nil, out.Plan != nil, out.Review != nil, out.NeedEvidence != nil} {
		if present {
			n++
		}
	}
	validRole := t.Role == planning.Analyst && (out.Tool != nil || out.Verification != nil) || t.Role == planning.Planner && (out.Plan != nil || out.NeedEvidence != nil) || t.Role == planning.Reviewer && out.Review != nil
	if n != 1 || !validRole {
		return out, fmt.Errorf("%w: exactly one role-appropriate action required", planning.ErrAgentModelOutput)
	}
	// References are sets. Ollama's grammar does not reliably enforce
	// uniqueItems; remove identical repetitions, never unknown references.
	if out.Verification != nil {
		for i := range out.Verification.Decisions {
			out.Verification.Decisions[i].EvidenceIDs = uniqueAgentReferences(out.Verification.Decisions[i].EvidenceIDs)
		}
	}
	if out.Plan != nil {
		for i := range out.Plan.Plans {
			out.Plan.Plans[i].EvidenceIDs = uniqueAgentReferences(out.Plan.Plans[i].EvidenceIDs)
		}
	}
	return out, nil
}

// The bundle contains the currently citable excerpts. Keep past requests and
// paging/error metadata for tool continuity without repeating all source text.
func compactAgentObservations(in []planning.ToolObservation) []planning.ToolObservation {
	out := make([]planning.ToolObservation, 0, len(in))
	for _, o := range in {
		copy := o
		copy.Result.Candidates = nil
		for _, c := range o.Result.Candidates {
			copy.Result.Candidates = append(copy.Result.Candidates, planning.Candidate{File: c.File, Symbol: c.Symbol})
		}
		out = append(out, copy)
	}
	return out
}

func uniqueAgentReferences(ids []string) []string {
	out := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		if !seen[id] {
			out = append(out, id)
			seen[id] = true
		}
	}
	return out
}

func unreadAgentSource(t planning.AgentTurn) (string, int) {
	read := map[string]bool{}
	for _, o := range t.Observations {
		if (o.Request.Name == "read_source" || o.Request.Name == "read_function" || o.Request.Name == "inspect_source") && o.Result.Error == "" && len(o.Result.Candidates) > 0 {
			read[o.Request.File] = true
		}
	}
	for i, c := range t.Bundle.Changes {
		for _, candidate := range c.Candidates {
			if !read[candidate.File] {
				return candidate.File, i
			}
		}
	}
	return "", 0
}

func agentSourceLine(t planning.AgentTurn, file string) int {
	for _, change := range t.Bundle.Changes {
		for _, candidate := range change.Candidates {
			if candidate.File != file {
				continue
			}
			for _, e := range candidate.Evidence {
				if strings.HasPrefix(e.Location, file+":") {
					line, err := strconv.Atoi(strings.TrimPrefix(e.Location, file+":"))
					if err == nil && line > 0 {
						return max(1, line-4)
					}
				}
			}
		}
	}
	return 1
}

func bindAgentTargets(schema map[string]any, action, field string, t planning.AgentTurn, plans bool) {
	if choices, ok := schema["oneOf"].([]any); ok {
		for _, choice := range choices {
			bindAgentTargets(choice.(map[string]any), action, field, t, plans)
		}
		return
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return
	}
	value, ok := props[action].(map[string]any)
	if !ok {
		return
	}
	array := value["properties"].(map[string]any)[field].(map[string]any)
	items := []any{}
	for _, change := range t.Bundle.Changes {
		for _, c := range change.Candidates {
			if plans {
				accepted := false
				for _, d := range t.Decisions {
					if d.ChangeType == change.ChangeType && d.ElementID == change.ElementID && d.File == c.File && d.Symbol == c.Symbol && d.Verdict != planning.VerdictReject {
						accepted = true
					}
				}
				if !accepted {
					continue
				}
			}
			item := schemaType(reflect.TypeOf(planning.CandidateDecision{}))
			if plans {
				item = schemaType(reflect.TypeOf(planning.FilePlan{}))
			}
			p := item["properties"].(map[string]any)
			for key, val := range map[string]string{"change_type": string(change.ChangeType), "element_id": change.ElementID, "file": c.File, "symbol": c.Symbol} {
				p[key] = map[string]any{"type": "string", "enum": []string{val}}
			}
			ids := []string{}
			for _, e := range c.Evidence {
				ids = append(ids, e.ID)
			}
			p["evidence_ids"] = map[string]any{"type": "array", "items": map[string]any{"type": "string", "enum": ids}, "minItems": 1, "maxItems": 8, "uniqueItems": true}
			if plans {
				p["module"] = map[string]any{"type": "string", "enum": []string{c.Module}}
				p["layer"] = map[string]any{"type": "string", "enum": []string{c.Layer}}
				actions := []string{string(c.SuggestedAction)}
				if c.SuggestedAction != "VERIFY" {
					actions = append(actions, "VERIFY")
				}
				if c.SuggestedAction != "UNRESOLVED" {
					actions = append(actions, "UNRESOLVED")
				}
				p["action"] = map[string]any{"type": "string", "enum": actions}
			} else {
				p["verdict"] = map[string]any{"type": "string", "enum": []string{"RELEVANT", "WEAK", "REJECT"}}
			}
			items = append(items, item)
		}
	}
	if len(items) > 0 {
		array["items"] = map[string]any{"oneOf": items}
	}
}

func constrainActionArray(schema map[string]any, action, field string, count int) {
	if choices, ok := schema["oneOf"].([]any); ok {
		for _, choice := range choices {
			constrainActionArray(choice.(map[string]any), action, field, count)
		}
		return
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return
	}
	value, ok := props[action].(map[string]any)
	if !ok {
		return
	}
	array := value["properties"].(map[string]any)[field].(map[string]any)
	array["minItems"] = count
	array["maxItems"] = count
}

func constrainActionEvidence(schema map[string]any, action, field string, ids []string) {
	if choices, ok := schema["oneOf"].([]any); ok {
		for _, choice := range choices {
			constrainActionEvidence(choice.(map[string]any), action, field, ids)
		}
		return
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return
	}
	value, ok := props[action].(map[string]any)
	if !ok {
		return
	}
	array := value["properties"].(map[string]any)[field].(map[string]any)
	evidence := array["items"].(map[string]any)["properties"].(map[string]any)["evidence_ids"].(map[string]any)
	if len(ids) > 0 {
		evidence["items"] = map[string]any{"type": "string", "enum": ids}
		evidence["minItems"] = 1
		evidence["uniqueItems"] = true
	}
}

func agentActionSchema(role planning.AgentRole) (map[string]any, error) {
	allowed := map[string]any{}
	switch role {
	case planning.Analyst:
		toolChoices := []any{}
		for _, name := range []string{"search_source", "read_function", "find_callers", "read_source", "inspect_source"} {
			props := map[string]any{"name": map[string]any{"type": "string", "enum": []string{name}}, "change_index": map[string]any{"type": "integer", "minimum": 0}}
			required := []string{"name", "change_index"}
			if name == "search_source" {
				props["offset"] = map[string]any{"type": "integer", "minimum": 0, "maximum": 10000}
				props["query"] = map[string]any{"type": "string", "minLength": 1, "maxLength": 200}
				required = append(required, "query")
			} else if name == "read_source" || name == "inspect_source" {
				props["file"] = map[string]any{"type": "string", "minLength": 1, "maxLength": 500}
				props["query"] = map[string]any{"type": "string", "maxLength": 200}
				props["start_line"] = map[string]any{"type": "integer", "minimum": 1}
				required = append(required, "file")
				if name == "inspect_source" {
					required = append(required, "query")
				}
			} else {
				props["file"] = map[string]any{"type": "string", "minLength": 1}
				props["symbol"] = map[string]any{"type": "string", "minLength": 1}
				required = append(required, "file", "symbol")
			}
			toolChoices = append(toolChoices, map[string]any{"type": "object", "properties": props, "required": required, "additionalProperties": false})
		}
		allowed["tool"] = map[string]any{"oneOf": toolChoices}
		allowed["verification"] = schemaType(reflect.TypeOf(planning.VerificationResponse{}))
	case planning.Planner:
		allowed["need_evidence"] = schemaType(reflect.TypeOf(planning.EvidenceRequest{}))
		allowed["plan"] = schemaType(reflect.TypeOf(planning.LayerPlanningResponse{}))
	case planning.Reviewer:
		issues := map[string]any{"type": "array", "items": map[string]any{"type": "string", "minLength": 1, "maxLength": 600}, "minItems": 1, "maxItems": 5}
		approved := map[string]any{"type": "object", "properties": map[string]any{"approved": map[string]any{"type": "boolean", "enum": []bool{true}}, "issues": map[string]any{"type": "array", "maxItems": 0, "items": map[string]any{"type": "string"}}}, "required": []string{"approved", "issues"}, "additionalProperties": false}
		needsReview := map[string]any{"type": "object", "properties": map[string]any{"approved": map[string]any{"type": "boolean", "enum": []bool{false}}, "issues": issues, "revision_target": map[string]any{"type": "string", "enum": []string{"analyst", "planner"}}}, "required": []string{"approved", "issues"}, "additionalProperties": false}
		allowed["review"] = map[string]any{"oneOf": []any{approved, needsReview}}
	default:
		return nil, errors.New("unknown agent role")
	}
	keys := make([]string, 0, len(allowed))
	for key := range allowed {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	choices := make([]any, 0, len(keys))
	for _, key := range keys {
		choices = append(choices, map[string]any{"type": "object", "properties": map[string]any{key: allowed[key]}, "required": []string{key}, "additionalProperties": false})
	}
	if len(choices) == 1 {
		return choices[0].(map[string]any), nil
	}
	return map[string]any{"oneOf": choices}, nil
}

func agentSearchedRepository(t planning.AgentTurn) bool {
	for _, o := range t.Observations {
		if o.Request.Name == "search_source" && o.Result.Error == "" {
			return true
		}
	}
	return false
}

func schemaType(t reflect.Type) map[string]any {
	if t.Kind() == reflect.Pointer {
		return schemaType(t.Elem())
	}
	switch t.Kind() {
	case reflect.Struct:
		props := map[string]any{}
		required := []string{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			tag := strings.Split(f.Tag.Get("json"), ",")
			props[tag[0]] = schemaType(f.Type)
			if f.Type.Kind() == reflect.Slice {
				limit := 20
				switch f.Name {
				case "Steps", "ProposedChanges", "AcceptanceChecks":
					limit = 8
				case "Issues", "Questions":
					limit = 5
				case "EvidenceIDs":
					limit = 8
				}
				props[tag[0]].(map[string]any)["maxItems"] = limit
			}
			if f.Name == "ProposedChanges" || f.Name == "AcceptanceChecks" {
				props[tag[0]].(map[string]any)["minItems"] = 1
			}
			if len(tag) == 1 || f.Name == "Symbol" || f.Name == "ProposedChanges" || f.Name == "AcceptanceChecks" {
				required = append(required, tag[0])
			}
		}
		return map[string]any{"type": "object", "properties": props, "required": required, "additionalProperties": false}
	case reflect.Slice:
		return map[string]any{"type": "array", "items": schemaType(t.Elem())}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int:
		return map[string]any{"type": "integer"}
	default:
		return map[string]any{"type": "string", "maxLength": 600}
	}
}

func agentInspectionQuery(t planning.AgentTurn, index int) string {
	c := t.Bundle.Changes[index]
	query := strings.TrimSpace(c.ElementName)
	if query == "" {
		query = c.ElementID
	}
	r := []rune(query)
	for len(string(r)) > 200 {
		r = r[:len(r)-1]
	}
	return string(r)
}
