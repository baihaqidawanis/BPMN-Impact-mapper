package llm

import (
	"context"
	"net/http"
	"sort"
	"strings"
)

// InstalledModelNames reads metadata only; no weights are loaded or installed.
func (m *OllamaAgent) InstalledModelNames(ctx context.Context) ([]string, error) {
	var tags struct {
		Models []struct {
			Name         string   `json:"name"`
			RemoteHost   string   `json:"remote_host"`
			RemoteModel  string   `json:"remote_model"`
			Capabilities []string `json:"capabilities"`
		} `json:"models"`
	}
	if err := m.request(ctx, http.MethodGet, "/api/tags", nil, &tags); err != nil {
		return nil, err
	}
	result := []string{}
	seen := map[string]bool{}
	for _, model := range tags.Models {
		if model.Name == "" || seen[model.Name] || model.RemoteHost != "" || model.RemoteModel != "" || strings.HasSuffix(model.Name, ":cloud") {
			continue
		}
		// Older runtimes omit capabilities; Prepare still verifies selection.
		allowed := len(model.Capabilities) == 0
		for _, c := range model.Capabilities {
			if c == "completion" {
				allowed = true
			}
		}
		if allowed {
			result = append(result, model.Name)
			seen[model.Name] = true
		}
	}
	sort.Strings(result)
	return result, nil
}
