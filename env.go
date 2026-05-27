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
	UserIsPaying           bool
	UserIsPayingKnown      bool
}

func DetectEnv() Env {
	localDir := os.Getenv("APIFY_LOCAL_STORAGE_DIR")
	if localDir == "" {
		localDir = filepath.Join(".", "storage")
	}

	userIsPaying, userIsPayingKnown := parseOptionalBool(os.Getenv("APIFY_USER_IS_PAYING"))

	return Env{
		LocalStorageDir:        localDir,
		ApifyToken:             os.Getenv("APIFY_TOKEN"),
		ActorRunID:             Coalesce(os.Getenv("ACTOR_RUN_ID"), os.Getenv("APIFY_ACTOR_RUN_ID")),
		ActorInputKey:          Coalesce(os.Getenv("ACTOR_INPUT_KEY"), "INPUT"),
		ActorDefaultDatasetID:  Coalesce(os.Getenv("ACTOR_DEFAULT_DATASET_ID"), os.Getenv("APIFY_DEFAULT_DATASET_ID"), "default"),
		ActorDefaultKeyValueID: Coalesce(os.Getenv("ACTOR_DEFAULT_KEY_VALUE_STORE_ID"), os.Getenv("APIFY_DEFAULT_KEY_VALUE_STORE_ID"), "default"),
		IsAtHome:               os.Getenv("APIFY_IS_AT_HOME") == "1",
		UserIsPaying:           userIsPaying,
		UserIsPayingKnown:      userIsPayingKnown,
	}
}

func (env Env) UsingApifyAPI() bool {
	return env.IsAtHome && env.ApifyToken != "" && env.ActorRunID != ""
}

type ResultLimitDecision struct {
	EffectiveMaxResults int
	Limited             bool
	Message             string
}

func ResolveResultLimit(env Env, requestedMaxResults, freeUserMaxResults int) ResultLimitDecision {
	decision := ResultLimitDecision{
		EffectiveMaxResults: requestedMaxResults,
	}

	if freeUserMaxResults <= 0 {
		return decision
	}

	if env.UserIsPayingKnown && env.UserIsPaying {
		return decision
	}

	if !env.UserIsPayingKnown && !env.IsAtHome {
		return decision
	}

	if requestedMaxResults <= 0 || requestedMaxResults > freeUserMaxResults {
		decision.EffectiveMaxResults = freeUserMaxResults
		decision.Limited = true
		decision.Message = "Free users are limited to preview results. Upgrade to a paid Apify plan for full output."
	}

	return decision
}
func parseOptionalBool(value string) (bool, bool) {
	switch normalizeEnvBool(value) {
	case "1", "true", "yes", "y", "on":
		return true, true
	case "0", "false", "no", "n", "off":
		return false, true
	default:
		return false, false
	}
}
