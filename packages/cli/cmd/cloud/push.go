package cloud

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type UploadManager struct {
	syncClient *SyncClient
	maxRetries int
	retryDelay time.Duration
}

func NewUploadManager(syncClient *SyncClient) *UploadManager {
	return &UploadManager{
		syncClient: syncClient,
		maxRetries: 3,
		retryDelay: 2 * time.Second,
	}
}

func (um *UploadManager) UploadRun(ctx context.Context, runID string) error {
	var lastErr error

	for attempt := 0; attempt < um.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(um.retryDelay):
			}
		}

		err := um.syncClient.PushRun(ctx, runID)
		if err == nil {
			return nil
		}

		lastErr = err

		if !um.isRetryable(err) {
			return err
		}
	}

	return fmt.Errorf("upload failed after %d attempts: %w", um.maxRetries, lastErr)
}

func (um *UploadManager) UploadBatch(ctx context.Context, runIDs []string) (*UploadResult, error) {
	result := &UploadResult{
		Total:      len(runIDs),
		Successful: 0,
		Failed:     0,
		Errors:     make(map[string]error),
	}

	for _, runID := range runIDs {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		err := um.UploadRun(ctx, runID)
		if err != nil {
			result.Failed++
			result.Errors[runID] = err
		} else {
			result.Successful++
		}
	}

	return result, nil
}

func (um *UploadManager) ValidateRun(runID string) error {
	recordingPath := filepath.Join(".sentra-lab", "recordings", runID)

	if _, err := os.Stat(recordingPath); os.IsNotExist(err) {
		return fmt.Errorf("run not found: %s", runID)
	}

	metadataPath := filepath.Join(recordingPath, "metadata.json")
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		return fmt.Errorf("metadata not found for run: %s", runID)
	}

	recordingFile := filepath.Join(recordingPath, "recording.zstd")
	if _, err := os.Stat(recordingFile); os.IsNotExist(err) {
		return fmt.Errorf("recording file not found for run: %s", runID)
	}

	info, err := os.Stat(recordingFile)
	if err != nil {
		return err
	}

	if info.Size() == 0 {
		return fmt.Errorf("recording file is empty for run: %s", runID)
	}

	return nil
}

func (um *UploadManager) isRetryable(err error) bool {
	return true
}

type UploadResult struct {
	Total      int
	Successful int
	Failed     int
	Errors     map[string]error
}

func (ur *UploadResult) String() string {
	return fmt.Sprintf("Uploaded %d/%d runs (%d failed)", ur.Successful, ur.Total, ur.Failed)
}
