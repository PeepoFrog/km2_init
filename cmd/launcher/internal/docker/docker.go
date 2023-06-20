package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

type DockerClient struct {
	Cli *client.Client
}

func NewDockerClient() (*DockerClient, error) {
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	return &DockerClient{Cli: client}, err
}

func (DC *DockerClient) VerifyDockerInstallation(ctx context.Context) error {
	// Try to ping the Docker daemon to check if it's running
	_, err := DC.Cli.Ping(context.Background())
	if err != nil {
		log.Println("Error pinging Docker daemon:", err)
		return err
	}

	// If we got here, Docker is installed and running
	log.Println("Docker is installed and running!")
	return nil
}

func (DC *DockerClient) PullImage(ctx context.Context, imageName string) error {
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

func (DC *DockerClient) GetFileFromContainer(ctx context.Context, filePathOnHostMachine, filePathOnContainer, containerID string) error {
	rc, _, err := DC.Cli.CopyFromContainer(context.Background(), containerID, filePathOnContainer)
	if err != nil {
		return err
	}
	defer rc.Close()

	contents, err := io.ReadAll(rc)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filePathOnHostMachine, contents, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (DC *DockerClient) SendFileToContainer(ctx context.Context, filePathOnHostMachine, filePathOnContainer, containerID string) error {
	// Open the file
	file, err := os.Open(filePathOnHostMachine)
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
	err = cli.CopyToContainer(ctx, containerID, filePathOnContainer, tarReader, copyOptions)
	if err != nil {
		return err
	}
	return nil
}

// required to send file into container
func addFileToTar(fileInfo os.FileInfo, file io.Reader, tarWriter *tar.Writer) error {
	// Create a new tar header
	header := &tar.Header{
		Name: fileInfo.Name(),
		Mode: int64(fileInfo.Mode()),
		Size: fileInfo.Size(),
	} // Write the header to the tar archive
	if err := tarWriter.WriteHeader(header); err != nil {
		log.Println(err)
		return err
	}
	// Copy the file data to the tar archive
	if _, err := io.Copy(tarWriter, file); err != nil {
		log.Println(err)
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
		log.Println(err)
		return err
	}
	attachOptions := types.ExecStartCheck{
		Detach: false,
		Tty:    false,
	}
	respConn, err := DC.Cli.ContainerExecAttach(context.Background(), resp.ID, attachOptions)
	if err != nil {
		log.Println(err)
		return err
	}
	defer respConn.Close()
	// Capture the output from the container
	output, err := io.ReadAll(respConn.Reader)
	if err != nil {
		log.Println(err)
		return err
	}
	// Wait for the execution to complete
	waitResponse, err := DC.Cli.ContainerExecInspect(context.Background(), resp.ID)
	if err != nil {
		log.Println(err)
		return err
	}
	if waitResponse.ExitCode != 0 {
		err = fmt.Errorf("package installation failed: %s", string(output))
		log.Println(err)
		return err
	} else {
		log.Println("Package installed successfully")
	}
	return nil
}
func (DC *DockerClient) ExecCommandInContainerInDetachMode(containerID string, command []string) ([]byte, error) {
	execCreateResponse, err := DC.Cli.ContainerExecCreate(context.Background(), containerID, types.ExecConfig{
		Cmd:          command,
		AttachStdout: false,
		AttachStderr: false,
		Detach:       true,
	})
	if err != nil {
		return nil, err
	}
	execAttachConfig := types.ExecStartCheck{}
	resp, err := DC.Cli.ContainerExecAttach(context.Background(), execCreateResponse.ID, execAttachConfig)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer resp.Close()

	// Read the output
	output, err := io.ReadAll(resp.Reader)
	if err != nil {
		log.Println(err)
		return output, err
	}
	return output, err
}

func (DC *DockerClient) ExecCommandInContainer(containerID string, command []string) ([]byte, error) {
	execCreateResponse, err := DC.Cli.ContainerExecCreate(context.Background(), containerID, types.ExecConfig{
		Cmd:          command,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return nil, err
	}
	execAttachConfig := types.ExecStartCheck{}
	resp, err := DC.Cli.ContainerExecAttach(context.Background(), execCreateResponse.ID, execAttachConfig)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer resp.Close()

	// Read the output
	output, err := io.ReadAll(resp.Reader)
	if err != nil {
		log.Println(err)
		return output, err
	}
	return output, err
}

// check if containers with same names existing, if yes delete
func (DC *DockerClient) CheckForContainersName(ctx context.Context, containerNameToCheck string) (bool, error) {
	containers, err := DC.Cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		log.Println(err)
		return true, err
	}
	for _, c := range containers {
		for _, name := range c.Names {
			if name == `/`+containerNameToCheck {
				log.Printf("container %v detected \n", name)
				return true, err
			}
		}
	}
	return false, err
}

// Stoping and deleting container
func (DC *DockerClient) StopAndDeleteContainer(ctx context.Context, containerNameToStop string) error {
	log.Printf("stoping %v container... \n", containerNameToStop)

	err := DC.Cli.ContainerStop(ctx, containerNameToStop, container.StopOptions{})
	if err != nil {
		log.Println(err)
		return err
	}
	log.Printf("deleting %v container... \n", containerNameToStop)
	err = DC.Cli.ContainerRemove(ctx, containerNameToStop, types.ContainerRemoveOptions{})
	if err != nil {
		log.Println(err)
		return err
	}
	log.Printf("container %v deleted \n", containerNameToStop)
	return err
}

func (DC *DockerClient) InitAndCreateContainer(ctx context.Context, containerCofig *container.Config, networkConfig *network.NetworkingConfig, hostConfig *container.HostConfig, containerName string) error {
	resp, err := DC.Cli.ContainerCreate(ctx, containerCofig, hostConfig, networkConfig, nil, containerName)
	if err != nil {
		log.Println(err)
		return err
	}
	// Start the container
	if err := DC.Cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Println(err)
		return err
	}
	log.Printf("'%s' container started successfully! ID: %s\n", containerName, resp.ID)
	return err

}

func (DC *DockerClient) CheckAndCreateNetwork(ctx context.Context, networkName string) error {
	networklist, err := DC.Cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		log.Println(err)
		return err
	}
	check := true
	for _, network := range networklist {
		if network.Name == networkName {
			check = false
			log.Printf("network %s already exists\n", networkName)
			return err
		}
	}
	if check {
		log.Println("creating network")
		_, err := DC.Cli.NetworkCreate(ctx, networkName, types.NetworkCreate{})
		if err != nil {
			log.Fatalf("Unable to create Docker network: %v", err)
			return err
		}
	}
	return err
}

// dont need yet
func replaceConfigFile(filePath string, oldString string, newString string) error {
	// Read the contents of the file
	contents, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	// Replace the old string with the new string
	modifiedContents := strings.ReplaceAll(string(contents), oldString, newString)

	// Write the modified contents back to the file
	err = ioutil.WriteFile(filePath, []byte(modifiedContents), 0644)
	if err != nil {
		return err
	}

	return nil
}
