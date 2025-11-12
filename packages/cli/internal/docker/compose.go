package docker

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type ComposeManager struct {
	client    *Client
	services  []ServiceDefinition
	network   string
	projectID string
}

func NewComposeManager(client *Client, projectID string) *ComposeManager {
	return &ComposeManager{
		client:    client,
		services:  []ServiceDefinition{},
		network:   fmt.Sprintf("sentra-lab-%s", projectID),
		projectID: projectID,
	}
}

func (cm *ComposeManager) AddService(service ServiceDefinition) {
	cm.services = append(cm.services, service)
}

func (cm *ComposeManager) StartAll(ctx context.Context) error {
	if err := cm.createNetwork(ctx); err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(cm.services))

	for _, service := range cm.services {
		wg.Add(1)
		go func(svc ServiceDefinition) {
			defer wg.Done()

			if err := cm.startService(ctx, svc); err != nil {
				errChan <- fmt.Errorf("service %s failed: %w", svc.Name, err)
			}
		}(service)
	}

	wg.Wait()
	close(errChan)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to start services: %v", errors)
	}

	return nil
}

func (cm *ComposeManager) StopAll(ctx context.Context) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(cm.services))

	for _, service := range cm.services {
		wg.Add(1)
		go func(svc ServiceDefinition) {
			defer wg.Done()

			containerName := cm.getContainerName(svc.Name)
			containers, err := cm.client.ListContainers(ctx)
			if err != nil {
				errChan <- err
				return
			}

			for _, container := range containers {
				if container.Name == containerName || container.Name == "/"+containerName {
					if err := cm.client.StopContainer(ctx, container.ID, 10*time.Second); err != nil {
						errChan <- fmt.Errorf("failed to stop %s: %w", svc.Name, err)
						return
					}

					if err := cm.client.RemoveContainer(ctx, container.ID); err != nil {
						errChan <- fmt.Errorf("failed to remove %s: %w", svc.Name, err)
						return
					}
				}
			}
		}(service)
	}

	wg.Wait()
	close(errChan)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to stop services: %v", errors)
	}

	return nil
}

func (cm *ComposeManager) GetServiceStatus(ctx context.Context) (map[string]*ContainerStatus, error) {
	statuses := make(map[string]*ContainerStatus)

	containers, err := cm.client.ListContainers(ctx)
	if err != nil {
		return nil, err
	}

	for _, service := range cm.services {
		containerName := cm.getContainerName(service.Name)

		for _, container := range containers {
			if container.Name == containerName || container.Name == "/"+containerName {
				statuses[service.Name] = container
				break
			}
		}
	}

	return statuses, nil
}

func (cm *ComposeManager) startService(ctx context.Context, service ServiceDefinition) error {
	if err := cm.client.PullImage(ctx, service.Image); err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	containerName := cm.getContainerName(service.Name)

	containers, err := cm.client.ListContainers(ctx)
	if err != nil {
		return err
	}

	for _, container := range containers {
		if container.Name == containerName || container.Name == "/"+containerName {
			if container.Running {
				return nil
			}

			if err := cm.client.RemoveContainer(ctx, container.ID); err != nil {
				return fmt.Errorf("failed to remove existing container: %w", err)
			}
		}
	}

	config := &ContainerConfig{
		Name:        containerName,
		Image:       service.Image,
		Ports:       service.Ports,
		Environment: service.Environment,
		Volumes:     service.Volumes,
		Memory:      service.Memory,
		CPUs:        service.CPUs,
	}

	containerID, err := cm.client.CreateContainer(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	if err := cm.client.StartContainer(ctx, containerID); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	return nil
}

func (cm *ComposeManager) createNetwork(ctx context.Context) error {
	return nil
}

func (cm *ComposeManager) getContainerName(serviceName string) string {
	return fmt.Sprintf("sentra-lab-%s-%s", cm.projectID, serviceName)
}

type ServiceDefinition struct {
	Name        string
	Image       string
	Ports       map[string]int
	Environment map[string]string
	Volumes     []string
	DependsOn   []string
	Memory      int64
	CPUs        float64
	HealthCheck *HealthCheck
}

type HealthCheck struct {
	Test     []string
	Interval time.Duration
	Timeout  time.Duration
	Retries  int
}