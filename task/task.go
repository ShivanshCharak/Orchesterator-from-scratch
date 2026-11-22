package task

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"

	dockerclient "cube/client"
)

type state int

const (
	Pending state = iota
	Scheduled
	Running
	Completed
	Failed
)

type Task struct {
	ID            uuid.UUID
	Name          string
	State         state
	Image         string
	Memory        int
	Disk          int
	ExposedPorts  nat.PortSet
	PortBinding   map[string]string
	RestartPolicy string
	startTime     time.Time
	endTime       time.Time
}

type TaskEvent struct {
	ID        uuid.UUID
	State     state
	Timestamp time.Time
	Task      Task
}

type Config struct {
	Name          string
	AttachStdin   bool
	AttachStdout  bool
	AttachStderr  bool
	ExposedPorts  nat.PortSet
	Cmd           []string
	Image         string
	Cpu           float64
	Memory        int64
	Disk          int64
	Env           []string
	RestartPolicy string
}

type Docker struct {
	Client *dockerclient.Client
	Config Config
}

type DockerResult struct {
	Error       error
	Action      string
	ContainerID string
	Result      string
}

func (d *Docker) Run() DockerResult {
	if d == nil || d.Client == nil {
		return DockerResult{Error: errors.New("Docker client is not initialized")}
	}
	if d.Config.Image == "" {
		return DockerResult{Error: errors.New("Docker image is required")}
	}

	ctx := context.Background()
	reader, err := d.Client.ImagePull(ctx, d.Config.Image, image.PullOptions{})
	if err != nil {
		return DockerResult{Error: err, Action: "pull"}
	}
	defer reader.Close()
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return DockerResult{Error: err, Action: "pull"}
	}

	ports := make(nat.PortSet, len(d.Config.ExposedPorts))
	for port := range d.Config.ExposedPorts {
		ports[port] = struct{}{}
	}
	containerConfig := &container.Config{
		AttachStdin:  d.Config.AttachStdin,
		AttachStdout: d.Config.AttachStdout,
		AttachStderr: d.Config.AttachStderr,
		ExposedPorts: ports,
		Cmd:          d.Config.Cmd,
		Image:        d.Config.Image,
		Env:          d.Config.Env,
	}

	restartPolicyMode := container.RestartPolicyMode(d.Config.RestartPolicy)
	if restartPolicyMode == "" {
		restartPolicyMode = container.RestartPolicyDisabled
	}
	hostConfig := &container.HostConfig{
		RestartPolicy:   container.RestartPolicy{Name: restartPolicyMode},
		PublishAllPorts: true,
		Resources: container.Resources{
			Memory:   d.Config.Memory,
			NanoCPUs: int64(d.Config.Cpu * 1_000_000_000),
		},
	}

	created, err := d.Client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, d.Config.Name)
	if err != nil {
		log.Printf("Error creating container using image %s: %v\n", d.Config.Image, err)
		return DockerResult{Error: err}
	}
	if err := d.Client.ContainerStart(ctx, created.ID, container.StartOptions{}); err != nil {
		log.Printf("Error starting container %s: %v\n", created.ID, err)
		return DockerResult{Error: err}
	}

	out, err := d.Client.ContainerLogs(ctx, created.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		log.Printf("Error getting logs for container %s: %v\n", created.ID, err)
		return DockerResult{Error: err}
	}
	defer out.Close()
	if _, err := stdcopy.StdCopy(os.Stdout, os.Stderr, out); err != nil {
		return DockerResult{Error: err, Action: "logs", ContainerID: created.ID}
	}

	return DockerResult{Action: "start", ContainerID: created.ID, Result: "success"}
}

func (d *Docker) Stop(id string) DockerResult {
	log.Printf("Attempting to stop the container %s", id)
	ctx := context.Background()
	if err := d.Client.ContainerStop(ctx, id, container.StopOptions{}); err != nil {
		log.Printf("Error stopping container %s: %v\n", id, err)
		return DockerResult{Error: err}
	}
	if err := d.Client.ContainerRemove(ctx, id, container.RemoveOptions{
		RemoveVolumes: true,
		RemoveLinks:   false,
		Force:         false,
	}); err != nil {
		log.Printf("Error removing container %s: %v\n", id, err)
		return DockerResult{Error: err}
	}
	return DockerResult{Action: "stop", ContainerID: id, Result: "success"}
}
