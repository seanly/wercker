package main

import (
	"os"
	"testing"

	"github.com/codegangsta/cli"
	"github.com/fsouza/go-dockerclient"
)

var (
	globalFlags   = flagsFor(GlobalFlags)
	pipelineFlags = flagsFor(PipelineFlags, WerckerInternalFlags)
	emptyFlags    = []cli.Flag{}
)

// DockerOrSkip checks for a docker container and skips the test
// if one is not available
func DockerOrSkip(t *testing.T) *DockerClient {
	if os.Getenv("SKIP_DOCKER_TEST") == "true" {
		t.Skip("$SKIP_DOCKER_TEST=true, skipping test")
		return nil
	}

	client, err := NewDockerClient(minimalDockerOptions())
	err = client.Ping()
	if err != nil {
		t.Skip("Docker not available, skipping test")
		return nil
	}
	return client
}

func emptyEnv() *Environment {
	return NewEnvironment()
}

func emptyPipelineOptions() *PipelineOptions {
	return &PipelineOptions{GlobalOptions: &GlobalOptions{}}
}

func minimalDockerOptions() *DockerOptions {
	opts := &DockerOptions{GlobalOptions: &GlobalOptions{}}
	guessAndUpdateDockerOptions(opts, NewEnvironment(os.Environ()...))
	return opts
}

type containerRemover struct {
	*docker.Container
	client *DockerClient
}

func tempBusybox(client *DockerClient) (*containerRemover, error) {
	_, err := client.InspectImage("alpine")
	if err != nil {
		options := docker.PullImageOptions{
			Repository: "alpine",
			Tag:        "3.1",
		}

		err = client.PullImage(options, docker.AuthConfiguration{})
		if err != nil {
			return nil, err
		}
	}

	container, err := client.CreateContainer(
		docker.CreateContainerOptions{
			Name: "temp-busybox",
			Config: &docker.Config{
				Image:           "alpine:3.1",
				Tty:             false,
				OpenStdin:       true,
				Cmd:             []string{"/bin/sh"},
				AttachStdin:     true,
				AttachStdout:    true,
				AttachStderr:    true,
				NetworkDisabled: true,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return &containerRemover{Container: container, client: client}, nil
}

func (cc *containerRemover) Remove() {
	if cc == nil {
		return
	}
	cc.client.RemoveContainer(docker.RemoveContainerOptions{
		ID:            cc.Container.ID,
		RemoveVolumes: true,
	})
}
