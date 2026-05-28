package actorsdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultAPIWriteAttempts = 4

type Client struct {
	Env          Env
	HTTPClient   *http.Client
	datasetDir   string
	outputPath   string
	nextDatasetN int
}

func NewClient(env Env) *Client {
	datasetDir := filepath.Join(env.LocalStorageDir, "datasets", env.ActorDefaultDatasetID)
	outputPath := filepath.Join(env.LocalStorageDir, "key_value_stores", env.ActorDefaultKeyValueID, "OUTPUT.json")

	client := &Client{
		Env:          env,
		HTTPClient:   http.DefaultClient,
		datasetDir:   datasetDir,
		outputPath:   outputPath,
		nextDatasetN: 1,
	}

	if env.UsingApifyAPI() {
		return client
	}

	entries, err := os.ReadDir(datasetDir)
	if err != nil {
		return client
	}

	nextN := 1
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		base := strings.TrimSuffix(entry.Name(), ".json")
		value, err := strconv.Atoi(base)
		if err == nil && value >= nextN {
			nextN = value + 1
		}
	}
	client.nextDatasetN = nextN
	return client
}

func ReadInput[T any](client *Client) (T, error) {
	return ReadRecord[T](client, client.Env.ActorInputKey)
}

func ReadRecord[T any](client *Client, key string) (T, error) {
	var value T

	localPath := filepath.Join(client.Env.LocalStorageDir, "key_value_stores", client.Env.ActorDefaultKeyValueID, key+".json")
	body, err := os.ReadFile(localPath)
	if err == nil {
		if err := json.Unmarshal(body, &value); err != nil {
			return value, fmt.Errorf("invalid local %s.json: %w", key, err)
		}
		return value, nil
	}

	if !client.Env.UsingApifyAPI() {
		return value, fmt.Errorf("%s not found locally and no actor run/token available", key)
	}

	recordURL := fmt.Sprintf(
		"https://api.apify.com/v2/actor-runs/%s/key-value-store/records/%s?token=%s",
		client.Env.ActorRunID,
		url.PathEscape(key),
		url.QueryEscape(client.Env.ApifyToken),
	)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, recordURL, nil)
	if err != nil {
		return value, err
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return value, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return value, fmt.Errorf("%s API returned %d: %s", key, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return value, err
	}
	if err := json.Unmarshal(body, &value); err != nil {
		return value, fmt.Errorf("invalid API %s json: %w", key, err)
	}
	return value, nil
}

func (client *Client) SetOutput(value any) error {
	return client.SetRecord("OUTPUT", value)
}

func (client *Client) SetRecord(key string, value any) error {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}

	if client.Env.UsingApifyAPI() {
		recordURL := fmt.Sprintf(
			"https://api.apify.com/v2/actor-runs/%s/key-value-store/records/%s?token=%s&contentType=application%%2Fjson",
			client.Env.ActorRunID,
			url.PathEscape(key),
			url.QueryEscape(client.Env.ApifyToken),
		)

		for attempt := 1; attempt <= defaultAPIWriteAttempts; attempt++ {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, recordURL, bytes.NewReader(body))
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.HTTPClient.Do(req)
			if err != nil {
				if attempt < defaultAPIWriteAttempts {
					time.Sleep(apiWriteRetryDelay(attempt))
					continue
				}
				return err
			}

			bodySnippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}

			if shouldRetryAPIWriteStatus(resp.StatusCode) && attempt < defaultAPIWriteAttempts {
				time.Sleep(apiWriteRetryDelay(attempt))
				continue
			}

			return fmt.Errorf("%s API returned %d: %s", key, resp.StatusCode, strings.TrimSpace(string(bodySnippet)))
		}
	}

	path := filepath.Join(client.Env.LocalStorageDir, "key_value_stores", client.Env.ActorDefaultKeyValueID, key+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

func PushData[T any](client *Client, rows []T) error {
	if len(rows) == 0 {
		return nil
	}

	if client.Env.UsingApifyAPI() {
		body, err := json.Marshal(rows)
		if err != nil {
			return err
		}

		datasetURL := fmt.Sprintf(
			"https://api.apify.com/v2/actor-runs/%s/dataset/items?token=%s",
			client.Env.ActorRunID,
			url.QueryEscape(client.Env.ApifyToken),
		)

		for attempt := 1; attempt <= defaultAPIWriteAttempts; attempt++ {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, datasetURL, bytes.NewReader(body))
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.HTTPClient.Do(req)
			if err != nil {
				if attempt < defaultAPIWriteAttempts {
					time.Sleep(apiWriteRetryDelay(attempt))
					continue
				}
				return err
			}

			bodySnippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}

			if shouldRetryAPIWriteStatus(resp.StatusCode) && attempt < defaultAPIWriteAttempts {
				time.Sleep(apiWriteRetryDelay(attempt))
				continue
			}

			return fmt.Errorf("dataset API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(bodySnippet)))
		}
	}

	if err := os.MkdirAll(client.datasetDir, 0o755); err != nil {
		return err
	}

	for _, row := range rows {
		path := filepath.Join(client.datasetDir, fmt.Sprintf("%09d.json", client.nextDatasetN))
		client.nextDatasetN++

		body, err := json.MarshalIndent(row, "", "\t")
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, body, 0o644); err != nil {
			return err
		}
	}

	return nil
}

func shouldRetryAPIWriteStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= 500
}

func apiWriteRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	return time.Duration(attempt*attempt) * 200 * time.Millisecond
}
