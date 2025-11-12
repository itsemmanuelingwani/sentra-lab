package docker

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type Client struct {
	cli *client.Client
}

func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &Client{cli: cli}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.cli.Ping(ctx)
	if err != nil {
		return fmt.Errorf("docker daemon not reachable: %w", err)
	}
	return nil
}

func (c *Client) PullImage(ctx context.Context, imageName string) error {
	reader, err := c.cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer reader.Close()

	_, err = io.Copy(io.Discard, reader)
	return err
}

func (c *Client) CreateContainer(ctx context.Context, config *ContainerConfig) (string, error) {
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}

	for containerPort, hostPort := range config.Ports {
		port, err := nat.NewPort("tcp", containerPort)
		if err != nil {
			return "", fmt.Errorf("invalid port %s: %w", containerPort, err)
		}

		portBindings[port] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: fmt.Sprintf("%d", hostPort),
			},
		}
		exposedPorts[port] = struct{}{}
	}

	envVars := []string{}
	for key, value := range config.Environment {
		envVars = append(envVars, fmt.Sprintf("%s=%s", key, value))
	}

	containerConfig := &container.Config{
		Image:        config.Image,
		Env:          envVars,
		ExposedPorts: exposedPorts,
	}

	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Binds:        config.Volumes,
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
	}

	if config.Memory > 0 {
		hostConfig.Memory = config.Memory
	}

	if config.CPUs > 0 {
		hostConfig.NanoCPUs = int64(config.CPUs * 1e9)
	}

	networkConfig := &network.NetworkingConfig{}

	resp, err := c.cli.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, config.Name)
	if err != nil {
		return "", fmt.Errorf("failed to create container %s: %w", config.Name, err)
	}

	return resp.ID, nil
}

func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	if err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container %s: %w", containerID, err)
	}
	return nil
}

func (c *Client) StopContainer(ctx context.Context, containerID string, timeout time.Duration) error {
	timeoutSeconds := int(timeout.Seconds())
	if err := c.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeoutSeconds}); err != nil {
		return fmt.Errorf("failed to stop container %s: %w", containerID, err)
	}
	return nil
}

func (c *Client) RemoveContainer(ctx context.Context, containerID string) error {
	if err := c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove container %s: %w", containerID, err)
	}
	return nil
}

func (c *Client) GetContainerLogs(ctx context.Context, containerID string, tail int) (string, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       fmt.Sprintf("%d", tail),
	}

	reader, err := c.cli.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return "", fmt.Errorf("failed to get logs for container %s: %w", containerID, err)
	}
	defer reader.Close()

	logs, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(logs), nil
}

func (c *Client) StreamContainerLogs(ctx context.Context, containerID string, output io.Writer) error {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: true,
	}

	reader, err := c.cli.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return fmt.Errorf("failed to stream logs for container %s: %w", containerID, err)
	}
	defer reader.Close()

	_, err = io.Copy(output, reader)
	return err
}

func (c *Client) GetContainerStatus(ctx context.Context, containerID string) (*ContainerStatus, error) {
	inspect, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}

	status := &ContainerStatus{
		ID:      inspect.ID,
		Name:    inspect.Name,
		State:   inspect.State.Status,
		Running: inspect.State.Running,
	}

	if inspect.State.StartedAt != "" {
		startedAt, _ := time.Parse(time.RFC3339Nano, inspect.State.StartedAt)
		status.StartedAt = startedAt
		status.Uptime = time.Since(startedAt)
	}

	if inspect.State.Health != nil {
		status.Health = inspect.State.Health.Status
	}

	return status, nil
}

func (c *Client) ListContainers(ctx context.Context) ([]*ContainerStatus, error) {
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var statuses []*ContainerStatus
	for _, ctr := range containers {
		status := &ContainerStatus{
			ID:      ctr.ID,
			Name:    ctr.Names[0],
			State:   ctr.State,
			Running: ctr.State == "running",
		}

		if ctr.State == "running" {
			status.StartedAt = time.Unix(ctr.Created, 0)
			status.Uptime = time.Since(status.StartedAt)
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

func (c *Client) GetContainerStats(ctx context.Context, containerID string) (*ContainerStats, error) {
	stats, err := c.cli.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats for container %s: %w", containerID, err)
	}
	defer stats.Body.Close()

	var v types.StatsJSON
	if err := io.ReadAll(stats.Body); err != nil {
		return nil, err
	}

	return &ContainerStats{
		CPUPercent:    calculateCPUPercent(&v),
		MemoryUsage:   v.MemoryStats.Usage,
		MemoryLimit:   v.MemoryStats.Limit,
		MemoryPercent: calculateMemoryPercent(&v),
		NetworkRx:     calculateNetworkRx(&v),
		NetworkTx:     calculateNetworkTx(&v),
	}, nil
}

func (c *Client) Close() error {
	return c.cli.Close()
}

type ContainerConfig struct {
	Name        string
	Image       string
	Ports       map[string]int
	Environment map[string]string
	Volumes     []string
	Memory      int64
	CPUs        float64
}

type ContainerStatus struct {
	ID        string
	Name      string
	State     string
	Running   bool
	Health    string
	StartedAt time.Time
	Uptime    time.Duration
}

type ContainerStats struct {
	CPUPercent    float64
	MemoryUsage   uint64
	MemoryLimit   uint64
	MemoryPercent float64
	NetworkRx     uint64
	NetworkTx     uint64
}

func calculateCPUPercent(stats *types.StatsJSON) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)

	if systemDelta > 0 && cpuDelta > 0 {
		return (cpuDelta / systemDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}

	return 0.0
}

func calculateMemoryPercent(stats *types.StatsJSON) float64 {
	if stats.MemoryStats.Limit > 0 {
		return float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit) * 100.0
	}
	return 0.0
}

func calculateNetworkRx(stats *types.StatsJSON) uint64 {
	var rx uint64
	for _, network := range stats.Networks {
		rx += network.RxBytes
	}
	return rx
}

func calculateNetworkTx(stats *types.StatsJSON) uint64 {
	var tx uint64
	for _, network := range stats.Networks {
		tx += network.TxBytes
	}
	return tx
}