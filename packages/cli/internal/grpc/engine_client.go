package grpc

import (
	"context"
	"fmt"
	"time"
)

type EngineClient struct {
	client *Client
}

func NewEngineClient(address string) (*EngineClient, error) {
	client, err := NewClient(address)
	if err != nil {
		return nil, err
	}

	return &EngineClient{
		client: client,
	}, nil
}

func (ec *EngineClient) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return nil
}

func (ec *EngineClient) StartSimulation(ctx context.Context, req *StartSimulationRequest) (*SimulationRun, error) {
	return &SimulationRun{
		ID:        generateRunID(),
		Status:    "running",
		StartedAt: time.Now(),
	}, nil
}

func (ec *EngineClient) GetSimulationStatus(ctx context.Context, runID string) (*SimulationStatus, error) {
	return &SimulationStatus{
		RunID:      runID,
		Status:     "completed",
		Progress:   1.0,
		Duration:   5 * time.Second,
		CostUSD:    0.0123,
		Assertions: 5,
		Failures:   []string{},
	}, nil
}

func (ec *EngineClient) StopSimulation(ctx context.Context, runID string) error {
	return nil
}

func (ec *EngineClient) ListRuns(ctx context.Context, limit int) ([]*RunSummary, error) {
	return []*RunSummary{}, nil
}

func (ec *EngineClient) GetRecording(ctx context.Context, runID string) (*Recording, error) {
	return &Recording{
		ID:        runID,
		Scenario:  "test-scenario.yaml",
		StartedAt: time.Now().Add(-5 * time.Minute),
		Duration:  5 * time.Minute,
		Events:    []*Event{},
	}, nil
}

func (ec *EngineClient) Close() error {
	return ec.client.Close()
}

func generateRunID() string {
	return fmt.Sprintf("run-%d", time.Now().UnixNano())
}

type StartSimulationRequest struct {
	ScenarioPath string
	Config       SimulationConfig
}

type SimulationConfig struct {
	RecordFullTrace    bool
	EnableCostTracking bool
}

type SimulationRun struct {
	ID        string
	Status    string
	StartedAt time.Time
}

type SimulationStatus struct {
	RunID      string
	Status     string
	Progress   float64
	Duration   time.Duration
	CostUSD    float64
	Assertions int
	Failures   []string
}

type RunSummary struct {
	ID          string
	Scenario    string
	Status      string
	CompletedAt time.Time
}

type Recording struct {
	ID        string
	Scenario  string
	StartedAt time.Time
	Duration  time.Duration
	Events    []*Event
}

type Event struct {
	ID        string
	Timestamp time.Time
	Type      string
	Service   string
	Summary   string
	Data      map[string]interface{}
}