package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/sentra-lab/cli/internal/utils"
)

type SyncClient struct {
	logger  *utils.Logger
	token   string
	baseURL string
	client  *http.Client
}

func NewSyncClient(logger *utils.Logger, token string) *SyncClient {
	return &SyncClient{
		logger:  logger,
		token:   token,
		baseURL: "https://api.sentra.dev/v1",
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (sc *SyncClient) PushRun(ctx context.Context, runID string) error {
	recordingPath := filepath.Join(".sentra-lab", "recordings", runID)

	metadataPath := filepath.Join(recordingPath, "metadata.json")
	metadata, err := os.ReadFile(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	recordingFile := filepath.Join(recordingPath, "recording.zstd")
	recording, err := os.ReadFile(recordingFile)
	if err != nil {
		return fmt.Errorf("failed to read recording: %w", err)
	}

	payload := map[string]interface{}{
		"run_id":    runID,
		"metadata":  json.RawMessage(metadata),
		"recording": recording,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/runs", sc.baseURL), bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sc.token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := sc.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}

func (sc *SyncClient) PullRun(ctx context.Context, runID string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/runs/%s", sc.baseURL, runID), nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sc.token))

	resp, err := sc.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("download failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	var payload struct {
		RunID     string          `json:"run_id"`
		Metadata  json.RawMessage `json:"metadata"`
		Recording []byte          `json:"recording"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	recordingPath := filepath.Join(".sentra-lab", "recordings", runID)
	if err := os.MkdirAll(recordingPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	metadataPath := filepath.Join(recordingPath, "metadata.json")
	if err := os.WriteFile(metadataPath, payload.Metadata, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	recordingFile := filepath.Join(recordingPath, "recording.zstd")
	if err := os.WriteFile(recordingFile, payload.Recording, 0644); err != nil {
		return fmt.Errorf("failed to write recording: %w", err)
	}

	return nil
}

func (sc *SyncClient) ListTeamRuns(ctx context.Context, limit int) ([]*CloudRun, error) {
	url := fmt.Sprintf("%s/runs?limit=%d", sc.baseURL, limit)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sc.token))

	resp, err := sc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	var response struct {
		Runs []*CloudRun `json:"runs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Runs, nil
}

func (sc *SyncClient) Sync(ctx context.Context) (*SyncStats, error) {
	stats := &SyncStats{}

	localRuns, err := sc.getLocalRuns()
	if err != nil {
		return nil, fmt.Errorf("failed to get local runs: %w", err)
	}

	cloudRuns, err := sc.ListTeamRuns(ctx, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to list cloud runs: %w", err)
	}

	cloudRunMap := make(map[string]*CloudRun)
	for _, run := range cloudRuns {
		cloudRunMap[run.ID] = run
	}

	for _, runID := range localRuns {
		if _, exists := cloudRunMap[runID]; !exists {
			if err := sc.PushRun(ctx, runID); err != nil {
				sc.logger.Warn(fmt.Sprintf("Failed to upload %s: %v", runID, err))
				continue
			}
			stats.Uploaded++
		}
	}

	localRunMap := make(map[string]bool)
	for _, runID := range localRuns {
		localRunMap[runID] = true
	}

	for _, run := range cloudRuns {
		if !localRunMap[run.ID] {
			if err := sc.PullRun(ctx, run.ID); err != nil {
				sc.logger.Warn(fmt.Sprintf("Failed to download %s: %v", run.ID, err))
				continue
			}
			stats.Downloaded++
		}
	}

	return stats, nil
}

func (sc *SyncClient) getLocalRuns() ([]string, error) {
	recordingsDir := ".sentra-lab/recordings"

	if _, err := os.Stat(recordingsDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(recordingsDir)
	if err != nil {
		return nil, err
	}

	var runIDs []string
	for _, entry := range entries {
		if entry.IsDir() {
			runIDs = append(runIDs, entry.Name())
		}
	}

	return runIDs, nil
}

type CloudRun struct {
	ID         string    `json:"id"`
	Scenario   string    `json:"scenario"`
	Status     string    `json:"status"`
	Duration   string    `json:"duration"`
	UploadedAt time.Time `json:"uploaded_at"`
	UploadedBy string    `json:"uploaded_by"`
}

type SyncStats struct {
	Uploaded   int
	Downloaded int
	Conflicts  int
}
