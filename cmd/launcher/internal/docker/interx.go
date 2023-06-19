package docker

import (
	"fmt"
	"log"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

type InterxManager struct {
	ContainerConfig        *container.Config
	SekaiHostConfig        *container.HostConfig
	SekaidNetworkingConfig *network.NetworkingConfig
	DockerClient           *DockerClient
}

func NewInterxManager(interxPort, dockerBaseImageName, volumeName, dockerNetworkName string, dockerClient *DockerClient) (*InterxManager, error) {
	natInterxPort, err := nat.NewPort("tcp", interxPort)
	if err != nil {
		log.Fatalln(err)
	}
	interxContainerConfig := &container.Config{
		Image:        dockerBaseImageName,
		Cmd:          []string{"/bin/bash"},
		Tty:          true,
		AttachStdin:  true,
		OpenStdin:    true,
		StdinOnce:    true,
		ExposedPorts: nat.PortSet{natInterxPort: struct{}{}},
	}
	interxNetworkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			dockerNetworkName: {},
		},
	}
	interxHostConfig := &container.HostConfig{
		Binds: []string{
			volumeName,
		},
		PortBindings: nat.PortMap{
			natInterxPort: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: interxPort}},
		},
		Privileged: true,
	}
	return &InterxManager{interxContainerConfig, interxHostConfig, interxNetworkingConfig, dockerClient}, err

}
func (i *InterxManager) RunInterxContainer(sekaidContainerName, inerxContainerName, rpc_port, grpc_port string) error {
	command := fmt.Sprintf(`interx init --rpc="http://%s:%s" --grpc="dns:///%s:%s" `, sekaidContainerName, rpc_port, sekaidContainerName, grpc_port)
	out, err := i.DockerClient.ExecCommandInContainer(inerxContainerName, []string{`bash`, `-c`, command})
	if err != nil {
		log.Println(err)
		return err
	}
	go func() {
		out, err = i.DockerClient.ExecCommandInContainer(inerxContainerName, []string{`bash`, `-c`, `interx start`})
		if err != nil {
			log.Println(err)
		}
		log.Println(string(out))
	}()
	log.Println("interx started")
	return err
}
