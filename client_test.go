package actorsdk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

type testInput struct {
	Name string `json:"name"`
}

type testRow struct {
	Value string `json:"value"`
}

func TestReadInputFromLocalStorage(t *testing.T) {
	root := t.TempDir()
	inputDir := filepath.Join(root, "key_value_stores", "default")
	if err := os.MkdirAll(inputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"name":"alpha"}`)
	if err := os.WriteFile(filepath.Join(inputDir, "INPUT.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}

	client := NewClient(Env{
		LocalStorageDir:        root,
		ActorInputKey:          "INPUT",
		ActorDefaultDatasetID:  "default",
		ActorDefaultKeyValueID: "default",
	})

	input, err := ReadInput[testInput](client)
	if err != nil {
		t.Fatal(err)
	}
	if input.Name != "alpha" {
		t.Fatalf("unexpected input value: %#v", input)
	}
}

func TestPushDataContinuesDatasetNumbering(t *testing.T) {
	root := t.TempDir()
	datasetDir := filepath.Join(root, "datasets", "default")
	if err := os.MkdirAll(datasetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(datasetDir, "000000007.json"), []byte(`{"value":"existing"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	client := NewClient(Env{
		LocalStorageDir:        root,
		ActorDefaultDatasetID:  "default",
		ActorDefaultKeyValueID: "default",
	})

	rows := []testRow{{Value: "next-a"}, {Value: "next-b"}}
	if err := PushData(client, rows); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"000000008.json", "000000009.json"} {
		path := filepath.Join(datasetDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected dataset file %s to exist: %v", name, err)
		}
	}
}

func TestSetOutputWritesOutputJSON(t *testing.T) {
	root := t.TempDir()
	client := NewClient(Env{
		LocalStorageDir:        root,
		ActorDefaultDatasetID:  "default",
		ActorDefaultKeyValueID: "default",
	})

	payload := map[string]any{"ok": true, "count": 2}
	if err := client.SetOutput(payload); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(root, "key_value_stores", "default", "OUTPUT.json")
	body, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["ok"] != true {
		t.Fatalf("unexpected output payload: %#v", parsed)
	}
}

func TestPushDataRetriesTransientDatasetAPIError(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/dataset/items") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if attempts.Add(1) < 3 {
			http.Error(w, "temporary upstream failure", http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	client := NewClient(Env{
		LocalStorageDir:        t.TempDir(),
		ActorDefaultDatasetID:  "default",
		ActorDefaultKeyValueID: "default",
		ActorRunID:             "run-123",
		ApifyToken:             "token-123",
		IsAtHome:               true,
	})
	client.HTTPClient = &http.Client{
		Transport: rewriteAPITransport{
			target: serverURL,
			base:   http.DefaultTransport,
		},
	}

	rows := []testRow{{Value: "retry-me"}}
	if err := PushData(client, rows); err != nil {
		t.Fatal(err)
	}
	if got := attempts.Load(); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}

type rewriteAPITransport struct {
	target *url.URL
	base   http.RoundTripper
}

func (t rewriteAPITransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = t.target.Scheme
	cloned.URL.Host = t.target.Host
	cloned.Host = t.target.Host
	return t.base.RoundTrip(cloned)
}
