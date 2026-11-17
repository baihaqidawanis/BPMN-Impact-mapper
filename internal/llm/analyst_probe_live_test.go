//go:build labdiagnostics

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
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/filestore"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

type probeTransport struct {
	base      http.RoundTripper
	root      string
	index     int
	hashes    []string
	releaseAt int
}

func (p *probeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Path != "/api/chat" {
		return p.base.RoundTrip(r)
	}
	raw, _ := io.ReadAll(r.Body)
	var body map[string]any
	json.Unmarshal(raw, &body)
	// Two requests per model; release that model after the last one.
	release := p.releaseAt
	if release == 0 {
		release = 1
	}
	if p.index < release {
		body["keep_alive"] = "2m"
	} else {
		body["keep_alive"] = 0
	}
	raw, _ = json.Marshal(body)
	r.Body = io.NopCloser(bytes.NewReader(raw))
	r.ContentLength = int64(len(raw))
	stem := filepath.Join(p.root, fmt.Sprintf("case-%d", p.index))
	os.WriteFile(stem+"-request.json", raw, 0600)
	parts, _ := json.Marshal([]any{body["messages"], body["format"], body["options"]})
	hash := sha256.Sum256(parts)
	p.hashes = append(p.hashes, hex.EncodeToString(hash[:]))
	response, err := p.base.RoundTrip(r)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	response.Body.Close()
	if err != nil {
		return nil, err
	}
	os.WriteFile(stem+"-response.json", data, 0600)
	response.Body = io.NopCloser(bytes.NewReader(data))
	p.index++
	return response, nil
}

type analystReplyModel struct {
	identity planning.ProviderIdentity
	reply    planning.AgentReply
}

func (m analystReplyModel) Identity() planning.ProviderIdentity { return m.identity }
func (m analystReplyModel) Turn(_ context.Context, t planning.AgentTurn) (planning.AgentReply, error) {
	if t.Role == planning.Analyst {
		return m.reply, nil
	}
	return planning.AgentReply{}, errors.New("probe intentionally stops before Planner")
}

type analystProbeResult struct {
	Model, Variant, InputSHA256, Status, Error string
	Seconds                                    float64
	Identity                                   planning.ProviderIdentity
	Reply                                      planning.AgentReply
}

// Explicit opt-in sends only bounded, previously approved source evidence from
// a selected checkpoint. No reference/ground-truth file or full repo is loaded.
func TestLivePairedAnalystProbe(t *testing.T) {
	if os.Getenv("BPMN_PAIRED_ANALYST_PROBE") != "1" {
		t.Skip("explicit lab probe opt-in required")
	}
	checkpoint := os.Getenv("BPMN_PROBE_CHECKPOINT")
	data, err := os.ReadFile(checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	var saved struct{ Session planning.AgentSession }
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	bundle := saved.Session.Bundle
	for i := range bundle.Changes {
		for j, c := range bundle.Changes[i].Candidates {
			anchors := []planning.EvidenceItem{}
			for _, e := range c.Evidence {
				if e.Type != "SOURCE_TOOL" {
					anchors = append(anchors, e)
				}
			}
			c.Evidence = anchors
			for _, o := range saved.Session.Observations {
				for _, found := range o.Result.Candidates {
					read := o.Request.Name == "inspect_source" || o.Request.Name == "read_source" || o.Request.Name == "read_function"
					if found.File == c.File && (found.Symbol == c.Symbol || read) {
						c.Evidence = planning.MergeAgentEvidence(c.Evidence, found.Evidence, read)
					}
				}
			}
			if len(c.Evidence) == 0 {
				t.Fatal("missing source evidence", c.File)
			}
			bundle.Changes[i].Candidates[j] = c
		}
	}
	bundle.ContextID, _ = planning.ComputeContextID(bundle)
	isolated := bundle
	isolated.ProcessContext = nil
	isolated.Changes = append([]planning.ChangeInput(nil), bundle.Changes...)
	isolated.Changes[0].Candidates = append([]planning.Candidate(nil), bundle.Changes[0].Candidates[:1]...)
	isolated.ContextID, _ = planning.ComputeContextID(isolated)
	variants := []struct {
		name   string
		bundle planning.EvidenceBundle
	}{{"isolated-source", isolated}, {"original-batch", bundle}}
	root := filepath.Join("..", "..", ".data", "model-probes", time.Now().UTC().Format("20060102T150405"))
	if err = os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	t.Log("Private diagnostic artifacts:", root)
	models := probeModelNames(t)
	hashes := []string{}
	results := []analystProbeResult{}
	for mi, name := range models {
		// Refuse to load a second model while another model is resident.
		response, err := (&http.Client{Timeout: 10 * time.Second}).Get("http://127.0.0.1:11435/api/ps")
		if err != nil {
			t.Fatal(err)
		}
		var running struct{ Models []struct{ Name string } }
		json.NewDecoder(response.Body).Decode(&running)
		response.Body.Close()
		if len(running.Models) > 0 {
			t.Fatal("another model remains resident; not loading concurrently", running.Models)
		}
		folder := filepath.Join(root, fmt.Sprint("model-", mi))
		os.MkdirAll(folder, 0700)
		transport := &probeTransport{base: &http.Transport{Proxy: nil}, root: folder}
		model, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: "http://127.0.0.1:11435", Model: name, ContextSize: 32768, MaxOutput: 4096, Profile: os.Getenv("BPMN_PROBE_PROFILE"), Client: &http.Client{Timeout: 150 * time.Second, Transport: transport}})
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		if err = model.Prepare(ctx); err != nil {
			cancel()
			t.Fatal(err)
		}
		// Release only the selected probe model, including after an early error.
		t.Cleanup(func() { releaseProbeModel(t, name) })
		for vi, v := range variants {
			start := time.Now()
			reply, callErr := model.Turn(ctx, planning.AgentTurn{Role: planning.Analyst, Bundle: v.bundle, RemainingTools: 0, RemainingRevisions: 0})
			result := analystProbeResult{Model: name, Variant: v.name, Seconds: time.Since(start).Seconds(), Identity: model.Identity(), Reply: reply}
			if callErr != nil {
				result.Status = "ADAPTER_REJECTED"
				result.Error = callErr.Error()
			} else {
				engine := planning.AgentEngine{Model: analystReplyModel{model.Identity(), reply}, Tools: noSourceTools{}, Store: filestore.AgentSessionStore{Root: filepath.Join(folder, fmt.Sprintf("validation-%d", vi))}, Budget: planning.AgentBudget{MaxTurns: 3}}
				checked, _ := engine.Run(context.Background(), "probe-validation", v.bundle)
				if checked.Role == planning.Planner {
					result.Status = "VALID_ANALYST_DECISIONS"
				} else {
					result.Status = "ENGINE_REJECTED"
					result.Error = strings.Join(checked.Feedback, "; ")
				}
			}
			if len(transport.hashes) > vi {
				result.InputSHA256 = transport.hashes[vi]
			}
			if mi == 0 {
				hashes = append(hashes, result.InputSHA256)
			} else if result.InputSHA256 != hashes[vi] {
				t.Error("paired input changed", v.name)
			}
			results = append(results, result)
			summary, _ := json.MarshalIndent(results, "", "  ")
			os.WriteFile(filepath.Join(root, "results.json"), summary, 0600)
			t.Logf("MODEL=%s variant=%s status=%s seconds=%.2f error=%s", name, v.name, result.Status, result.Seconds, result.Error)
			if reply.Verification != nil {
				for _, d := range reply.Verification.Decisions {
					t.Logf("DECISION %s %s %s", d.File, d.Verdict, d.Rationale)
				}
			}
		}
		cancel()
		releaseProbeModel(t, name)
	}
}

func probeModelNames(t *testing.T) []string {
	t.Helper()
	value := os.Getenv("BPMN_PROBE_MODELS")
	if value == "" {
		return []string{"qwen3.5:9b-q8_0", "bpmn-qwen-coder:30b-q4_k_m"}
	}
	models := []string{}
	seen := map[string]bool{}
	for _, v := range strings.Split(value, ",") {
		name := strings.TrimSpace(v)
		if name == "" || strings.Contains(strings.ToLower(name), ":cloud") || seen[name] {
			t.Fatal("explicit unique local model names required")
		}
		seen[name] = true
		models = append(models, name)
	}
	return models
}

func releaseProbeModel(t *testing.T, name string) {
	t.Helper()
	client := &http.Client{Timeout: 15 * time.Second}
	response, err := client.Get("http://127.0.0.1:11435/api/ps")
	if err != nil {
		t.Log("cannot inspect probe cleanup:", err)
		return
	}
	var running struct{ Models []struct{ Name string } }
	err = json.NewDecoder(response.Body).Decode(&running)
	response.Body.Close()
	if err != nil {
		t.Log("invalid cleanup metadata:", err)
		return
	}
	found := false
	for _, m := range running.Models {
		if m.Name == name {
			found = true
		} else {
			t.Log("another model became active; not unloading runtime")
			return
		}
	}
	if !found {
		return
	}
	body, _ := json.Marshal(map[string]any{"model": name, "keep_alive": 0, "stream": false})
	response, err = client.Post("http://127.0.0.1:11435/api/generate", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Log("probe unload failed:", err)
		return
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Log("probe unload status:", response.StatusCode)
	}
}
