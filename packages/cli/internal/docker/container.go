package docker

import (
	"context"
	"fmt"
	"time"
)

type Container struct {
	ID        string
	Name      string
	Image     string
	Status    string
	CreatedAt time.Time
	Ports     map[string]int
	client    *Client
}

func NewContainer(client *Client, id string) *Container {
	return &Container{
		ID:     id,
		client: client,
	}
}

func (c *Container) Start(ctx context.Context) error {
	return c.client.StartContainer(ctx, c.ID)
}

func (c *Container) Stop(ctx context.Context, timeout time.Duration) error {
	return c.client.StopContainer(ctx, c.ID, timeout)
}

func (c *Container) Restart(ctx context.Context) error {
	if err := c.Stop(ctx, 10*time.Second); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	return c.Start(ctx)
}

func (c *Container) Remove(ctx context.Context) error {
	return c.client.RemoveContainer(ctx, c.ID)
}

func (c *Container) Logs(ctx context.Context, tail int) (string, error) {
	return c.client.GetContainerLogs(ctx, c.ID, tail)
}

func (c *Container) GetStatus(ctx context.Context) (*ContainerStatus, error) {
	return c.client.GetContainerStatus(ctx, c.ID)
}

func (c *Container) GetStats(ctx context.Context) (*ContainerStats, error) {
	return c.client.GetContainerStats(ctx, c.ID)
}

func (c *Container) IsRunning(ctx context.Context) (bool, error) {
	status, err := c.GetStatus(ctx)
	if err != nil {
		return false, err
	}
	return status.Running, nil
}

func (c *Container) IsHealthy(ctx context.Context) (bool, error) {
	status, err := c.GetStatus(ctx)
	if err != nil {
		return false, err
	}
	return status.Health == "healthy", nil
}

func (c *Container) WaitUntilHealthy(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		status, err := c.GetStatus(ctx)
		if err != nil {
			return err
		}

		if !status.Running {
			return fmt.Errorf("container not running")
		}

		if status.Health == "healthy" || status.Health == "" {
			return nil
		}

		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("timeout waiting for container to be healthy")
}

func (c *Container) Exec(ctx context.Context, cmd []string) (string, error) {
	return "", fmt.Errorf("not implemented")
}