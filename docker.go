package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"io"
	"os"
	"strings"
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

func getDockerObjects() (*client.Client, context.Context) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	return cli, ctx
}

func (dc *simpleDockerContainer) getImage() error {
	out, _ := dc.client.ImageList(dc.ctx, types.ImageListOptions{})

	hasImage := false
	for _, o := range out {
		if o.RepoTags[0] == dc.imageName {
			hasImage = true
		}
	}
	if hasImage != true {
		fmt.Println("Failed to find image, pulling")
		reader, err := dc.client.ImagePull(dc.ctx, dc.imageName, types.ImagePullOptions{})
		if err != nil {
			panic(err)
		}
		io.Copy(os.Stdout, reader)
	}
	return nil
}

func (dc *simpleDockerContainer) startContainer() (string, error) {
	if dc.containerID == "" {
		if err := dc.getImage(); err != nil {
			return "", err
		}
		port, binding, err := nat.ParsePortSpecs([]string{dc.exposedPort})
		if err != nil {
			return "", err
		}

		config := &container.Config{
			Image: dc.imageName, ExposedPorts: nat.PortSet(port),
		}

		hostConfig := &container.HostConfig{PortBindings: binding,}

		resp, err := dc.client.ContainerCreate(dc.ctx, config, hostConfig, nil, "")
		if err != nil {
			return "", err
		}

		if err := dc.client.ContainerStart(dc.ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
			return "", err
		}

		dc.containerID = resp.ID
	}
	return dc.containerID, nil
}

func (dc *simpleDockerContainer) getContainerNetworkInfo() (string, string) {
	if dc.hostname == "" && dc.port == "" {
		containers, err := dc.client.ContainerList(dc.ctx, types.ContainerListOptions{})
		if err != nil {
			panic(err)
		}
		var hostname, port string
		for _, con := range containers {
			json, _ := dc.client.ContainerInspect(dc.ctx, con.ID)
			if con.ID == dc.containerID {
				for _, v := range json.NetworkSettings.NetworkSettingsBase.Ports {
					hostname, port = v[0].HostIP, v[0].HostPort
				}
			}
		}
		dc.hostname, dc.port = hostname, port
	}
	return dc.hostname, dc.port
}

func (dc *simpleDockerContainer) stopContainer() error {
	if err := dc.client.ContainerStop(dc.ctx, dc.containerID, nil); err != nil {
		return err
	}
	if err := dc.client.ContainerRemove(dc.ctx, dc.containerID, types.ContainerRemoveOptions{}); err != nil {
		return err
	}

	dc.hostname, dc.port, dc.containerID = "", "", ""
	return nil
}
