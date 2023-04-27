package docker

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type DockerClient struct {
	Cli *client.Client
}

func GetDockerClient() (*DockerClient, error) {
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	return &DockerClient{Cli: client}, err
}

func (DC *DockerClient) VerifyDockerInstallation(ctx context.Context) error {
	// Create a new Docker client instance
	// cli, err := client.NewClientWithOpts(client.FromEnv)
	// if err != nil {
	// 	fmt.Println("Error creating Docker client:", err)
	// 	return err
	// }

	// Try to ping the Docker daemon to check if it's running
	_, err := DC.Cli.Ping(context.Background())
	if err != nil {
		fmt.Println("Error pinging Docker daemon:", err)
		return err
	}

	// If we got here, Docker is installed and running
	fmt.Println("Docker is installed and running!")
	return nil
}
func (DC *DockerClient) PullImage(ctx context.Context, imageName string) error {

	// Create a Docker client
	// cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	// if err != nil {
	// 	log.Fatalf("Unable to create Docker client: %s", err)
	// }
	// defer cli.Close()
	options := types.ImagePullOptions{}
	reader, err := DC.Cli.ImagePull(ctx, imageName, options)
	if err != nil {
		return err
	}
	defer reader.Close()

	// TODO: Add buffer for reader. Pretify output
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return err
	}

	return nil
}
func (DC *DockerClient) RunContainer(ctx context.Context, image string) error {

	// cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	// if err != nil {
	// 	panic(err)
	// }
	// defer cli.Close()
	resp, err := DC.Cli.ContainerCreate(ctx, &container.Config{
		Image: image,
		Cmd:   []string{"echo", "hello world"},
		Tty:   false,
	}, nil, nil, nil, "")
	if err != nil {
		panic(err)
	}

	if err := DC.Cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	statusCh, errCh := DC.Cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	out, err := DC.Cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		panic(err)
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	return nil
}
