//go:build integration

package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
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
	out, err := dc.client.ImageList(dc.ctx, image.ListOptions{})
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	repoDigest := ""
	for _, o := range out {
		//fmt.Println(fmt.Printf("List of images %+v", o))
		if len(o.RepoTags) > 0 && o.RepoTags[0] == dc.imageName {
			fmt.Println(fmt.Printf("Image summary: %+v", o))
			repoDigest = o.RepoDigests[0]
		}
	}
	if repoDigest != "" {
		fmt.Println("Failed to find image, pulling")
		reader, err := dc.client.ImagePull(dc.ctx, repoDigest, image.PullOptions{})
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

		hostConfig := &container.HostConfig{PortBindings: binding}

		resp, err := dc.client.ContainerCreate(dc.ctx, config, hostConfig, nil, nil, "redis")
		if err != nil {
			return "", err
		}

		if err := dc.client.ContainerStart(dc.ctx, resp.ID, container.StartOptions{}); err != nil {
			return "", err
		}

		dc.containerID = resp.ID
	}
	return dc.containerID, nil
}

func (dc *simpleDockerContainer) getContainerNetworkInfo() (string, string) {
	if dc.hostname == "" && dc.port == "" {
		var hostname, port string
		json, _ := dc.client.ContainerInspect(dc.ctx, dc.containerID)
		if len(json.NetworkSettings.NetworkSettingsBase.Ports) != 1 {
			panic("Too many ports mapped")
		}
		for _, v := range json.NetworkSettings.NetworkSettingsBase.Ports {
			fmt.Println(fmt.Printf("Network info %+v", v))
			if len(v) > 0 {
				hostname, port = v[0].HostIP, v[0].HostPort
			}
		}

		dc.hostname, dc.port = hostname, port
	}
	return dc.hostname, dc.port
}

func (dc *simpleDockerContainer) stopContainer() error {
	if err := dc.client.ContainerStop(dc.ctx, dc.containerID, container.StopOptions{}); err != nil {
		return err
	}
	if err := dc.client.ContainerRemove(dc.ctx, dc.containerID, container.RemoveOptions{}); err != nil {
		return err
	}

	dc.hostname, dc.port, dc.containerID = "", "", ""
	return nil
}
