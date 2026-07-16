package executor

import (
	"context"
	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type DockerExecutor struct {
	cli *client.Client
}

func NewDockerExecutor() (*DockerExecutor, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerExecutor{cli}, nil
}

func (d *DockerExecutor) Execute(ctx context.Context, task Task) error {
	reader, err := d.cli.ImagePull(ctx, task.Image, types.ImagePullOptions{})
	if err != nil {
		return err
	}

	defer reader.Close()
	io.Copy(io.Discard, reader) // Wait for the download to finish

	resp, err := d.cli.ContainerCreate(ctx,
		&container.Config{
			Image: task.Image,
			Cmd:   task.Cmd,
		},
		&container.HostConfig{
			Resources: container.Resources{
				Memory:   int64(task.ResourceRequirement.RequiredSystemMemory),
				NanoCPUs: int64(task.ResourceRequirement.RequiredSystemCPU * 1e9),
			},
		},
		nil, nil, "",
	)
	if err != nil {
		return err
	}

	// Cleanup
	defer d.cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})

	// Start the container
	err = d.cli.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return err
	}

	// Wait for the container to stop running
	statusCh, errCh := d.cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		return err
	case <-statusCh:
		// Container exited successfully
	}

	// Fetch and demultiplex container logs directly to worker stdout/stderr
	out, err := d.cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return err
	}
	defer out.Close()

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	return nil
}
