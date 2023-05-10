package docker

import (
	"archive/tar"
	"bytes"
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

func (DC *DockerClient) SendFileToContainer(ctx context.Context, filePath, containerPath, containerID string) error {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	// Get file information
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}
	// Create a Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	// Create a tar writer
	var buf bytes.Buffer
	tarWriter := tar.NewWriter(&buf)
	// Add file to the tar archive
	err = addFileToTar(fileInfo, file, tarWriter)
	if err != nil {
		return err
	}
	// Close the tar writer
	err = tarWriter.Close()
	if err != nil {
		return err
	}
	// Get the tar archive content as a []byte
	tarContent := buf.Bytes()
	// Create a reader from the tar archive content
	tarReader := bytes.NewReader(tarContent)
	// Create a `types.CopyToContainerOptions` struct with the desired options
	copyOptions := types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: false,
	}
	// Use `CopyToContainer` to send the tar archive to the container
	err = cli.CopyToContainer(ctx, containerID, containerPath, tarReader, copyOptions)
	if err != nil {
		return err
	}
	return nil
}

func addFileToTar(fileInfo os.FileInfo, file io.Reader, tarWriter *tar.Writer) error {
	// Create a new tar header
	header := &tar.Header{
		Name: fileInfo.Name(),
		Mode: int64(fileInfo.Mode()),
		Size: fileInfo.Size(),
	} // Write the header to the tar archive
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}
	// Copy the file data to the tar archive
	if _, err := io.Copy(tarWriter, file); err != nil {
		return err
	}
	return nil
}

func (DC *DockerClient) InstallDebPackage(containerID, debDestPath string) error {
	installCmd := []string{"dpkg", "-i", debDestPath}
	execOptions := types.ExecConfig{
		Cmd:          installCmd,
		AttachStdout: true,
		AttachStderr: true,
	}
	resp, err := DC.Cli.ContainerExecCreate(context.Background(), containerID, execOptions)
	if err != nil {
		panic(err)
	}
	attachOptions := types.ExecStartCheck{
		Detach: false,
		Tty:    false,
	}
	respConn, err := DC.Cli.ContainerExecAttach(context.Background(), resp.ID, attachOptions)
	if err != nil {
		return err
	}
	defer respConn.Close()
	// Capture the output from the container
	output, err := io.ReadAll(respConn.Reader)
	if err != nil {
		return err
	}
	// Wait for the execution to complete
	waitResponse, err := DC.Cli.ContainerExecInspect(context.Background(), resp.ID)
	if err != nil {
		return err
	}
	if waitResponse.ExitCode != 0 {
		fmt.Printf("Package installation failed: %s\n", string(output))
	} else {
		fmt.Println("Package installed successfully")
	}
	return nil
}
