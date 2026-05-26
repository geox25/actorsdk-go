package actorsdk

import (
	"encoding/json"
	"os"
	"path/filepath"
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
