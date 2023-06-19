package docker

import (
	"fmt"
	"log"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

type SekaidManager struct {
	ContainerConfig        *container.Config
	SekaiHostConfig        *container.HostConfig
	SekaidNetworkingConfig *network.NetworkingConfig
	DockerClient           *DockerClient
}

func NewSekaidManager(grpcPort, rpcPort, dockerBaseImageName, volumeName, dockerNetworkName string, dockerClient *DockerClient) (*SekaidManager, error) {
	natGrcpPort, err := nat.NewPort("tcp", grpcPort)
	if err != nil {
		log.Println(err)
		return &SekaidManager{}, err
	}
	natRpcPort, err := nat.NewPort("tcp", rpcPort)
	if err != nil {
		log.Println(err)
		return &SekaidManager{}, err
	}

	sekaiContainerConfig := &container.Config{
		Image:       dockerBaseImageName,
		Cmd:         []string{"/bin/bash"},
		Tty:         true,
		AttachStdin: true,
		OpenStdin:   true,
		StdinOnce:   true,
		ExposedPorts: nat.PortSet{
			natGrcpPort: struct{}{},
			natRpcPort:  struct{}{}},
	}

	sekaiHostConfig := &container.HostConfig{
		Binds: []string{
			volumeName,
		},
		PortBindings: nat.PortMap{
			natGrcpPort: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: grpcPort}},
			natRpcPort:  []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: rpcPort}},
		},
		Privileged: true,
	}

	sekaidNetworkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			dockerNetworkName: {},
		},
	}
	return &SekaidManager{sekaiContainerConfig, sekaiHostConfig, sekaidNetworkingConfig, dockerClient}, err
}

func (s *SekaidManager) RunSekaidContainer(sekaiContainerName, sekaiNetworkName, sekaidHome, keyringBackend, rcpPort string) error {
	moniker := "M O N I K E R"
	mneminicDir := `~/mnemonics`

	// command := `sekaid init  --overwrite --chain-id=` + sekaiNetworkName + ` --home=` + SEKAID_HOME + ` "` + moniker + `"`
	command := fmt.Sprintf(`sekaid init  --overwrite --chain-id=%s --home=%s "%s"`, sekaiNetworkName, sekaidHome, moniker)
	log.Println(command)
	out, err := s.DockerClient.ExecCommandInContainer(sekaiContainerName, []string{`bash`, `-c`, command})
	if err != nil {
		log.Println(err)
		return err
	}

	command = fmt.Sprintf(`mkdir %s`, mneminicDir)
	log.Println(command)
	out, err = s.DockerClient.ExecCommandInContainer(sekaiContainerName, []string{`bash`, `-c`, command})
	if err != nil {
		log.Println(err)
		return err
	}

	command = fmt.Sprintf(`sekaid keys add "validator" --keyring-backend=%s --home=%s --output=json | jq .mnemonic > %s/sekai.mnemonic`, keyringBackend, sekaidHome, mneminicDir)
	log.Println(command)
	out, err = s.DockerClient.ExecCommandInContainer(sekaiContainerName, []string{`bash`, `-c`, command})
	if err != nil {
		log.Println(err)
		return err
	}

	command = fmt.Sprintf(`sekaid keys add "faucet" --keyring-backend=%s --home=%s --output=json | jq .mnemonic > %s/faucet.mnemonic`, keyringBackend, sekaidHome, mneminicDir)
	log.Println(command)
	out, err = s.DockerClient.ExecCommandInContainer(sekaiContainerName, []string{`bash`, `-c`, command})
	if err != nil {
		log.Println(err)
		return err
	}

	command = fmt.Sprintf(`sekaid add-genesis-account validator 150000000000000ukex,300000000000000test,2000000000000000000000000000samolean,1000000lol --keyring-backend=%v --home=%v`, keyringBackend, sekaidHome)
	log.Println(command)
	out, err = s.DockerClient.ExecCommandInContainer(sekaiContainerName, []string{`bash`, `-c`, command})
	if err != nil {
		log.Println(err)
		return err
	}

	command = fmt.Sprintf(`sekaid gentx-claim validator --keyring-backend=%s --moniker="GENESIS VALIDATOR" --home=%s`, keyringBackend, sekaidHome)
	log.Println(command)
	out, err = s.DockerClient.ExecCommandInContainer(sekaiContainerName, []string{`bash`, `-c`, command})
	if err != nil {
		log.Println(err)
		return err
	}
	log.Println(string(out))

	go func() {
		command := fmt.Sprintf(`sekaid start --rpc.laddr "tcp://0.0.0.0:%s" --home=%s`, rcpPort, sekaidHome)
		log.Println(command)
		out, err = s.DockerClient.ExecCommandInContainer(sekaiContainerName, []string{`bash`, `-c`, command})
		if err != nil {
			log.Println(err)
		}
	}()
	log.Println("sekai started")
	return err
}
