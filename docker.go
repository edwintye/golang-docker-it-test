//go:build integration

package main

import (
	"context"
	"fmt"
	"io"
	"net/netip"
	"os"
	"slices"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

type simpleDockerContainer struct {
	client      *client.Client
	ctx         context.Context
	imageName   string
	exposedPort string
	containerID string
	hostname    string
	port        string
}

type DockerContainerInterface interface {
	initialize(string, string) error
	getImage() error
	startContainer() (string, error)
	getContainerNetworkInfo() (string, string)
	stopContainer() error
}

func getDockerObjects() (*client.Client, context.Context) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	return cli, ctx
}

func (dc *simpleDockerContainer) initialize(imageName string, exposedPort string) error {
	dc.exposedPort = exposedPort
	if len(strings.Split(imageName, ":")) == 2 {
		dc.imageName = imageName
	} else {
		dc.imageName = imageName + ":latest"
	}
	dc.client, dc.ctx = getDockerObjects()
	_, err := dc.startContainer()
	if err != nil {
		return err
	}
	dc.getContainerNetworkInfo()
	return nil
}

func (dc *simpleDockerContainer) getImage() error {
	out, err := dc.client.ImageList(dc.ctx, client.ImageListOptions{})
	if err != nil {
		fmt.Printf("Failed to list images %s\n", err.Error())
		return err
	}

	repoDigest := ""
	for _, o := range out.Items {
		if len(o.RepoTags) > 0 && slices.Contains(o.RepoTags, dc.imageName) {
			repoDigest = o.RepoDigests[0]
		}
	}
	pullRef := repoDigest
	if repoDigest != "" {
		fmt.Println("Failed to find image but repo digest obtained, pulling")
	} else {
		fmt.Println("Failed to find image, pulling using the image name")
		pullRef = dc.imageName
	}
	reader, err := dc.client.ImagePull(dc.ctx, pullRef, client.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()
	_, _ = io.Copy(os.Stdout, reader)
	return nil
}

func (dc *simpleDockerContainer) startContainer() (string, error) {
	if dc.containerID == "" {
		if err := dc.getImage(); err != nil {
			return "", err
		}

		// network has no ParsePortSpecs equivalent; the only spec used is a bare port,
		// which ParsePort defaults to /tcp, bound to all interfaces with a random host port.
		p, err := network.ParsePort(dc.exposedPort)
		if err != nil {
			return "", err
		}

		config := &container.Config{
			Image:        dc.imageName,
			ExposedPorts: network.PortSet{p: {}},
		}

		hostConfig := &container.HostConfig{
			PortBindings: network.PortMap{p: {{HostIP: netip.IPv4Unspecified(), HostPort: ""}}},
		}

		resp, err := dc.client.ContainerCreate(dc.ctx, client.ContainerCreateOptions{
			Config:     config,
			HostConfig: hostConfig,
			Name:       "redis",
		})
		if err != nil {
			return "", err
		}

		if _, err := dc.client.ContainerStart(dc.ctx, resp.ID, client.ContainerStartOptions{}); err != nil {
			return "", err
		}

		dc.containerID = resp.ID
	}
	return dc.containerID, nil
}

func (dc *simpleDockerContainer) getContainerNetworkInfo() (string, string) {
	if dc.hostname == "" && dc.port == "" {
		var hostname, port string
		json, err := dc.client.ContainerInspect(dc.ctx, dc.containerID, client.ContainerInspectOptions{})
		if err != nil {
			panic(err)
		}
		ports := json.Container.NetworkSettings.Ports
		if len(ports) != 1 {
			panic("Too many ports mapped")
		}
		for _, v := range ports {
			fmt.Printf("Network info %+v\n", v)
			if len(v) > 0 {
				hostname, port = v[0].HostIP.String(), v[0].HostPort
			}
		}

		dc.hostname, dc.port = hostname, port
	}
	return dc.hostname, dc.port
}

func (dc *simpleDockerContainer) stopContainer() error {
	if _, err := dc.client.ContainerStop(dc.ctx, dc.containerID, client.ContainerStopOptions{}); err != nil {
		return err
	}
	if _, err := dc.client.ContainerRemove(dc.ctx, dc.containerID, client.ContainerRemoveOptions{}); err != nil {
		return err
	}

	dc.hostname, dc.port, dc.containerID = "", "", ""
	return nil
}
