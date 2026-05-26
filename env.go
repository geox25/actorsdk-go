package actorsdk

import (
	"os"
	"path/filepath"
)

type Env struct {
	LocalStorageDir        string
	ApifyToken             string
	ActorRunID             string
	ActorInputKey          string
	ActorDefaultDatasetID  string
	ActorDefaultKeyValueID string
	IsAtHome               bool
}

func DetectEnv() Env {
	localDir := os.Getenv("APIFY_LOCAL_STORAGE_DIR")
	if localDir == "" {
		localDir = filepath.Join(".", "storage")
	}

	return Env{
		LocalStorageDir:        localDir,
		ApifyToken:             os.Getenv("APIFY_TOKEN"),
		ActorRunID:             Coalesce(os.Getenv("ACTOR_RUN_ID"), os.Getenv("APIFY_ACTOR_RUN_ID")),
		ActorInputKey:          Coalesce(os.Getenv("ACTOR_INPUT_KEY"), "INPUT"),
		ActorDefaultDatasetID:  Coalesce(os.Getenv("ACTOR_DEFAULT_DATASET_ID"), os.Getenv("APIFY_DEFAULT_DATASET_ID"), "default"),
		ActorDefaultKeyValueID: Coalesce(os.Getenv("ACTOR_DEFAULT_KEY_VALUE_STORE_ID"), os.Getenv("APIFY_DEFAULT_KEY_VALUE_STORE_ID"), "default"),
		IsAtHome:               os.Getenv("APIFY_IS_AT_HOME") == "1",
	}
}

func (env Env) UsingApifyAPI() bool {
	return env.IsAtHome && env.ApifyToken != "" && env.ActorRunID != ""
}
