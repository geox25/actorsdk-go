# actorsdk-go

Minimal, stdlib-only Go runtime helpers for Apify actors written in Go.

Module path:

- `github.com/geox25/actorsdk-go`

What it handles:

- runtime env detection
- local `storage/` input/output/dataset I/O
- direct Apify API fallback for at-home runs
- tiny utility helpers used across lean actors

What it intentionally does not handle:

- crawling frameworks
- browser automation
- retries/backoff policies
- actor-specific API adapters
- hidden abstractions around HTTP clients

The goal is to keep the shared layer small enough that importing it is effectively free, while leaving scraping strategy and actor-specific dependencies to each actor.

## Minimal usage

```go
package main

import (
	"log"

	actorsdk "github.com/geox25/actorsdk-go"
)

type Input struct {
	Queries []string `json:"queries"`
}

func main() {
	env := actorsdk.DetectEnv()
	client := actorsdk.NewClient(env)

	input, err := actorsdk.ReadInput[Input](client)
	if err != nil {
		log.Fatal(err)
	}

	rows := []map[string]any{
		{"query_count": len(input.Queries)},
	}
	if err := actorsdk.PushData(client, rows); err != nil {
		log.Fatal(err)
	}

	if err := client.SetOutput(map[string]any{"ok": true}); err != nil {
		log.Fatal(err)
	}
}
```
