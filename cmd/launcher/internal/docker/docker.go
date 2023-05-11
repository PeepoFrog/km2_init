package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
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
		err = fmt.Errorf("package installation failed: %s", string(output))
		return err
	} else {
		fmt.Println("Package installed successfully")
	}
	return nil
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
		panic(err)
	}
	defer resp.Close()

	// Read the output
	output, err := io.ReadAll(resp.Reader)
	if err != nil {
		panic(err)
	}

	return output, nil
}

func (DC *DockerClient) InitScript(ctx context.Context, imagename, nameForSekaiContainer, nameForInerxContainer string) {
	fmt.Println("stop")
	DC.Cli.ContainerStop(ctx, nameForSekaiContainer, container.StopOptions{})
	DC.Cli.ContainerStop(ctx, nameForInerxContainer, container.StopOptions{})
	DC.Cli.ContainerRemove(ctx, nameForSekaiContainer, types.ContainerRemoveOptions{})
	DC.Cli.ContainerRemove(ctx, nameForInerxContainer, types.ContainerRemoveOptions{})
	fmt.Println("END STOP")

	config := &container.Config{
		Image:       imagename,
		Cmd:         []string{"/bin/bash"},
		Tty:         true,
		AttachStdin: true,
		OpenStdin:   true,
		StdinOnce:   true,
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			"testVolume:/data",
		},
		PortBindings: nat.PortMap{
			"9090/tcp":  []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "9090"}},
			"26657/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "26657"}},
		},
		Privileged: true,
	}

	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"bridge": {},
		},
	}

	// Create the container for sekaid
	resp, err := DC.Cli.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, nameForSekaiContainer)
	if err != nil {
		panic(err)
	}

	// Start the container
	if err := DC.Cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	fmt.Printf("Sekai Container started successfully! ID: %s\n", resp.ID)

	config = &container.Config{
		Image:       imagename,
		Cmd:         []string{"/bin/bash"},
		Tty:         true,
		AttachStdin: true,
		OpenStdin:   true,
		StdinOnce:   true,
	}

	hostConfig = &container.HostConfig{
		Binds: []string{
			"testVolume:/data",
		},
		PortBindings: nat.PortMap{
			"11000/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "11000"}},
		},
		Privileged: true,
	}

	// Create the container for interx
	resp, err = DC.Cli.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, nameForInerxContainer)
	if err != nil {
		panic(err)
	}

	// Start the container
	if err := DC.Cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

}

func (DC *DockerClient) RunBlockChain(ctx context.Context, sekaiContainerName, inerxContainerName string) {

	// out, err := DC.ExecCommandInContainer(sekaiContainerName, []string{`bash`, `-c`, `NETWORK_NAME="PEPEGENETWORK-1" && \
	// SEKAID_HOME=~/.sekaid-$NETWORK_NAME`})
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(string(out))
	NETWORK_NAME := `PEPEGENETWORK-1`
	SEKAID_HOME := `/root/.sekaid-` + NETWORK_NAME

	// `sekaid init --overwrite --chain-id=$NETWORK_NAME "PEPEGA NETWORK" --home=$SEKAID_HOME`
	command := `sekaid init --overwrite --chain-id=` + NETWORK_NAME + ` "PEPEGA NETWORK" --home=` + SEKAID_HOME
	fmt.Println("1", command)
	out, err := DC.ExecCommandInContainer(sekaiContainerName, []string{`bash`, `-c`, command})
	if err != nil {
		panic(err)
	}
	// fmt.Println(string(out))
	fmt.Println("2")
	out, err = DC.ExecCommandInContainer(sekaiContainerName, []string{`bash`, `-c`, `mkdir ~/mnemonics`})
	if err != nil {
		panic(err)
	}
	// fmt.Println(string(out))
	fmt.Println("3")
	out, err = DC.ExecCommandInContainer(sekaiContainerName, []string{`bash`, `-c`, `sekaid keys add "validator" --keyring-backend=test --home=` + SEKAID_HOME + ` --output=json | jq .mnemonic > ~/mnemonics/sekai.mnemonic
	`})
	if err != nil {
		panic(err)
	}
	// fmt.Println(string(out))
	fmt.Println("4")
	out, err = DC.ExecCommandInContainer(sekaiContainerName, []string{`bash`, `-c`, `sekaid keys add "faucet" --keyring-backend=test --home=` + SEKAID_HOME + ` --output=json | jq .mnemonic > ~/mnemonics/faucet.mnemonic
	`})
	if err != nil {
		panic(err)
	}
	// fmt.Println(string(out))
	fmt.Println("5")
	out, err = DC.ExecCommandInContainer(sekaiContainerName, []string{`bash`, `-c`, `sekaid add-genesis-account validator 150000000000000ukex,300000000000000test,2000000000000000000000000000samolean,1000000lol --keyring-backend=test  --home=` + SEKAID_HOME})
	if err != nil {
		panic(err)
	}
	// fmt.Println(string(out))
	fmt.Println("6")
	out, err = DC.ExecCommandInContainer(sekaiContainerName, []string{`bash`, `-c`, `sekaid gentx-claim validator --keyring-backend=test --moniker="GENESIS VALIDATOR" --home=` + SEKAID_HOME})
	if err != nil {
		panic(err)
	}
	// fmt.Println(string(out))

	// err = DC.GetFileFromContainer(ctx, `./config.toml`, `root/.sekaid-PEPEGENETWORK-1/config/config.toml`, sekaiContainerName)
	// if err != nil {
	// 	panic(err)
	// }
	// err = replaceConfigFile(`./config.toml`, `laddr = "tcp://127.0.0.1:26657"`, `laddr = "tcp://0.0.0.0:26657"`)
	// if err != nil {
	// 	panic(err)
	// }
	// err = DC.SendFileToContainer(ctx, `./config.toml`, `root/.sekaid-PEPEGENETWORK-1/config/`, sekaiContainerName)
	// if err != nil {
	// 	panic(err)
	// }
	// command = `sed -i 's/laddr = "tcp:\/\/127.0.0.1:26657"/laddr = "tcp:\/\/0.0.0.0:26657" ~/.sekaid-PEPEGENETWORK-1/config/config.toml`
	// fmt.Println("8", command)
	// out, err = DC.ExecCommandInContainer(sekaiContainerName, []string{`bash`, `-c`, command})
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(string(out))

	// command = `sed -i 's/laddr = "tcp:\/\/127.0.0.1:26657"/laddr = "tcp:\/\/0.0.0.0:26657"/' ~/.sekaid/config/config.toml`
	// fmt.Println("8", command)
	// out, err = DC.ExecCommandInContainer(sekaiContainerName, []string{`bash`, `-c`, command})
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(string(out))

	// command = `cat > /etc/systemd/system/sekai.service << EOL
	// [Unit]
	// Description=Local KIRA Test Network
	// After=network.target
	// [Service]
	// MemorySwapMax=0
	// Type=simple
	// User=root
	// WorkingDirectory=/root
	// ExecStart=/bin/sekaid start --trace --home=` + SEKAID_HOME + `
	// Restart=always
	// RestartSec=5
	// LimitNOFILE=4096
	// [Install]
	// WantedBy=default.target
	// EOL`
	// fmt.Println("9", command)
	// out, err = DC.ExecCommandInContainer(sekaiContainerName, []string{`bash`, `-c`, command})
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(string(out))
	// DC.SendFileToContainer(ctx, `./sekai.service`, `/etc/systemd/system/sekai.service`, sekaiContainerName)
	// fmt.Println("10")
	// out, err = DC.ExecCommandInContainer(sekaiContainerName, []string{`bash`, `-c`, `systemctl enable sekai
	// `})
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(string(out))
	// fmt.Println("11")
	// out, err = DC.ExecCommandInContainer(sekaiContainerName, []string{`bash`, `-c`, `systemctl start sekai
	// `})
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(string(out))
	fmt.Println("12")
	go func() {
		out, err = DC.ExecCommandInContainer(sekaiContainerName, []string{`bash`, `-c`, `sekaid start --rpc.laddr "tcp://0.0.0.0:26657" --home=root/.sekaid-PEPEGENETWORK-1`})
		if err != nil {
			panic(err)
		}
		// fmt.Println(string(out))
	}()

	// INTERAX START
	fmt.Println("13")

	out, err = DC.ExecCommandInContainer(inerxContainerName, []string{`bash`, `-c`, `DEFAULT_GRPC_PORT=9090 && \
	DEFAULT_RPC_PORT=26657 && \
	PING_TARGET="172.17.0.2"`})
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))
	DEFAULT_GRPC_PORT := `9090`
	DEFAULT_RPC_PORT := `26657`
	PING_TARGET := `172.17.0.2`
	fmt.Println("14")

	out, err = DC.ExecCommandInContainer(inerxContainerName, []string{`bash`, `-c`, `interx init --rpc="http://` + PING_TARGET + `:` + DEFAULT_RPC_PORT + `" --grpc="dns:///` + PING_TARGET + `:` + DEFAULT_GRPC_PORT + `" 
	`})
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))
	fmt.Println("15")

	out, err = DC.ExecCommandInContainer(inerxContainerName, []string{`bash`, `-c`, `interx start 
	`})
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))

	fmt.Println("DOOOOOOOOOONEEEEEEEEEEEEEEEEEE")

}
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
