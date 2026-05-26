package actorsdk

import "testing"

func TestDetectEnvDefaults(t *testing.T) {
	t.Setenv("APIFY_LOCAL_STORAGE_DIR", "")
	t.Setenv("APIFY_TOKEN", "")
	t.Setenv("ACTOR_RUN_ID", "")
	t.Setenv("APIFY_ACTOR_RUN_ID", "")
	t.Setenv("ACTOR_INPUT_KEY", "")
	t.Setenv("ACTOR_DEFAULT_DATASET_ID", "")
	t.Setenv("APIFY_DEFAULT_DATASET_ID", "")
	t.Setenv("ACTOR_DEFAULT_KEY_VALUE_STORE_ID", "")
	t.Setenv("APIFY_DEFAULT_KEY_VALUE_STORE_ID", "")
	t.Setenv("APIFY_IS_AT_HOME", "")

	env := DetectEnv()

	if env.LocalStorageDir != "./storage" && env.LocalStorageDir != "storage" {
		t.Fatalf("unexpected LocalStorageDir: %q", env.LocalStorageDir)
	}
	if env.ActorInputKey != "INPUT" {
		t.Fatalf("unexpected ActorInputKey: %q", env.ActorInputKey)
	}
	if env.ActorDefaultDatasetID != "default" {
		t.Fatalf("unexpected ActorDefaultDatasetID: %q", env.ActorDefaultDatasetID)
	}
	if env.ActorDefaultKeyValueID != "default" {
		t.Fatalf("unexpected ActorDefaultKeyValueID: %q", env.ActorDefaultKeyValueID)
	}
	if env.IsAtHome {
		t.Fatal("expected IsAtHome to be false")
	}
	if env.UsingApifyAPI() {
		t.Fatal("expected UsingApifyAPI to be false")
	}
}

func TestDetectEnvOverrideAndUsingApifyAPI(t *testing.T) {
	t.Setenv("APIFY_LOCAL_STORAGE_DIR", "/tmp/custom-storage")
	t.Setenv("APIFY_TOKEN", "token")
	t.Setenv("ACTOR_RUN_ID", "run-123")
	t.Setenv("ACTOR_INPUT_KEY", "CUSTOM_INPUT")
	t.Setenv("ACTOR_DEFAULT_DATASET_ID", "dataset-1")
	t.Setenv("ACTOR_DEFAULT_KEY_VALUE_STORE_ID", "store-1")
	t.Setenv("APIFY_IS_AT_HOME", "1")

	env := DetectEnv()

	if env.LocalStorageDir != "/tmp/custom-storage" {
		t.Fatalf("unexpected LocalStorageDir: %q", env.LocalStorageDir)
	}
	if env.ActorRunID != "run-123" {
		t.Fatalf("unexpected ActorRunID: %q", env.ActorRunID)
	}
	if env.ActorInputKey != "CUSTOM_INPUT" {
		t.Fatalf("unexpected ActorInputKey: %q", env.ActorInputKey)
	}
	if !env.UsingApifyAPI() {
		t.Fatal("expected UsingApifyAPI to be true")
	}
}
