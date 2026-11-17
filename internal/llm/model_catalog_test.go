package llm

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestCatalogListsOnlyInstalledLocalGenerationAliases(t *testing.T) {
	calls := 0
	m, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: "http://127.0.0.1:11435", Model: "fixture", Client: &http.Client{Transport: errorTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "GET" || r.URL.Path != "/api/tags" {
			t.Fatal("catalog sent non-metadata request", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"models":[{"name":"z-local","capabilities":["completion","thinking"]},{"name":"a-old"},{"name":"embed","capabilities":["embedding"]},{"name":"cloud-alias","remote_model":"remote"},{"name":"remote:cloud"},{"name":"z-local"}]}`))}, nil
	})}})
	if err != nil {
		t.Fatal(err)
	}
	names, err := m.InstalledModelNames(context.Background())
	if err != nil || calls != 1 || !reflect.DeepEqual(names, []string{"a-old", "z-local"}) {
		t.Fatal(names, calls, err)
	}
}
