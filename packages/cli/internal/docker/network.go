package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
)

type NetworkManager struct {
	client *Client
}

func NewNetworkManager(client *Client) *NetworkManager {
	return &NetworkManager{client: client}
}

func (nm *NetworkManager) CreateNetwork(ctx context.Context, name string) (string, error) {
	networks, err := nm.client.cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list networks: %w", err)
	}

	for _, network := range networks {
		if network.Name == name {
			return network.ID, nil
		}
	}

	response, err := nm.client.cli.NetworkCreate(ctx, name, types.NetworkCreate{
		Driver: "bridge",
		Labels: map[string]string{
			"created-by": "sentra-lab",
		},
	})

	if err != nil {
		return "", fmt.Errorf("failed to create network: %w", err)
	}

	return response.ID, nil
}

func (nm *NetworkManager) RemoveNetwork(ctx context.Context, networkID string) error {
	if err := nm.client.cli.NetworkRemove(ctx, networkID); err != nil {
		return fmt.Errorf("failed to remove network: %w", err)
	}
	return nil
}

func (nm *NetworkManager) ConnectContainer(ctx context.Context, networkID, containerID string) error {
	if err := nm.client.cli.NetworkConnect(ctx, networkID, containerID, nil); err != nil {
		return fmt.Errorf("failed to connect container to network: %w", err)
	}
	return nil
}

func (nm *NetworkManager) DisconnectContainer(ctx context.Context, networkID, containerID string) error {
	if err := nm.client.cli.NetworkDisconnect(ctx, networkID, containerID, false); err != nil {
		return fmt.Errorf("failed to disconnect container from network: %w", err)
	}
	return nil
}