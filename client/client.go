package client

import (
	"context"
	"errors"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// Client wraps the Docker Engine API client.
type Client struct {
	*client.Client
}

func New(options ...client.Opt) (*Client, error) {
	dockerClient, err := client.NewClientWithOpts(options...)
	if err != nil {
		return nil, err
	}
	return &Client{Client: dockerClient}, nil
}

func (cli *Client) ContainerCreate(
	ctx context.Context,
	config *container.Config,
	hostConfig *container.HostConfig,
	networkingConfig *network.NetworkingConfig,
	platform *v1.Platform,
	containerName string,
) (container.CreateResponse, error) {
	if cli == nil || cli.Client == nil {
		return container.CreateResponse{}, errors.New("Docker client is not initialized")
	}
	return cli.Client.ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, containerName)
}
