package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/google/go-github/github"
	"github.com/mrlutik/km2_init/cmd/launcher/internal/adapters"
	"github.com/mrlutik/km2_init/cmd/launcher/internal/cosign"
	"github.com/mrlutik/km2_init/cmd/launcher/internal/docker"
	"github.com/mrlutik/km2_init/cmd/launcher/internal/helpers"
	"golang.org/x/oauth2"
)

const (
	NETWORK_NAME          = "testnet-1"
	SEKAID_HOME           = "~/sekaid"
	KEYRING_BACKEND       = "test"
	DOCKER_IMAGE_NAME     = "ghcr.io/kiracore/docker/kira-base"
	DOCKER_IMAGE_VERSION  = "v0.13.11"
	DOCKER_NETWORK_NAME   = "kira_network"
	SEKAI_VERSION         = "latest" //or v0.3.16
	INTERX_VERSION        = "latest" //or v0.4.33
	SEKAID_CONTAINER_NAME = "sekaid"
	INTERX_CONTAINER_NAME = "interx"
	VOLUME_NAME           = "kira_volume"
	MNEMONIC_FOLDER       = "~/mnemonics"
	RPC_PORT              = 26657
	GRPC_PORT             = 9090
	INTERX_PORT           = 11000
)
const DockerImagePubKey = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE/IrzBQYeMwvKa44/DF/HB7XDpnE+
f+mU9F/Qbfq25bBWV2+NlYMJv3KvKHNtu3Jknt6yizZjUV4b8WGfKBzFYw==
-----END PUBLIC KEY-----`

func main() {
	dockerClient, err := docker.GetDockerClient()
	if err != nil {
		panic(err)
	}
	defer dockerClient.Cli.Close()
	launcherInterface := helpers.LauncherInterface(&helpers.Linux{})
	dockerBaseImageName := DOCKER_IMAGE_NAME + ":" + DOCKER_IMAGE_VERSION
	// dockerBaseImageName := "ubuntu"
	err = launcherInterface.PrivilageCheck()
	if err != nil {
		panic(err)
	}

	arch, platform := launcherInterface.CheckPlaform()
	fmt.Println(arch, platform)
	ctx := context.Background()
	err = dockerClient.VerifyDockerInstallation(ctx)
	defer dockerClient.Cli.Close()
	if err != nil {
		fmt.Println(err)
		fmt.Println("INSTALING DOCKER")
		if err = launcherInterface.InstallDocker(); err != nil {
			panic(err)
		}
		err = dockerClient.VerifyDockerInstallation(ctx)
		if err != nil {
			panic(err)
		}
	}

	fmt.Println("docker installed")
	err = dockerClient.PullImage(ctx, dockerBaseImageName)
	if err != nil {
		panic(err)
	}
	b, err := cosign.VerifyImageSignature(ctx, dockerBaseImageName, DockerImagePubKey)
	if err != nil {
		fmt.Println(b)
		panic(err)
	}
	fmt.Println(b)
	r := adapters.Repositories{}
	kiraRepos := []string{"sekai", "interx"}
	kiraGit := "KiraCore"
	for _, v := range kiraRepos {
		switch v {
		case "sekai":
			r.Set(kiraGit, v, SEKAI_VERSION)
		case "interx":
			r.Set(kiraGit, v, INTERX_VERSION)
		default:
			r.Set(kiraGit, v)
		}

	}
	fmt.Println(r.Get())
	fmt.Println(os.LookupEnv("GITHUB_TOKEN"))
	token := os.Getenv("GITHUB_TOKEN")
	r = adapters.Fetch(r, token)
	fmt.Println(r)
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	githubClient := github.NewClient(tc)
	debFileDestInContainer := "/"
	sekaiDebFileName := "sekai-linux-amd64.deb"
	interxDebFileName := "interx-linux-amd64.deb"

	fmt.Println("INIT SCRIPT")
	dockerClient.CheckForContainersName(ctx, SEKAID_CONTAINER_NAME, INTERX_CONTAINER_NAME)
	dockerClient.CreateNetwork(ctx, "kira_network")
	//creating sekaid container
	log.Println("Creating containers")
	log.Println("Creating sekaid container")
	dockerClient.InitAndCreateSekaidContainer(ctx, dockerBaseImageName, SEKAID_CONTAINER_NAME, DOCKER_NETWORK_NAME)
	log.Println("Creating interx container")
	dockerClient.InitAndCreateInterxContainer(ctx, dockerBaseImageName, INTERX_CONTAINER_NAME, DOCKER_NETWORK_NAME)
	// goto F
	adapters.DownloadBinaryFromRepo(ctx, githubClient, "KiraCore", "sekai", sekaiDebFileName, SEKAI_VERSION)
	adapters.DownloadBinaryFromRepo(ctx, githubClient, "KiraCore", "interx", interxDebFileName, INTERX_VERSION)
	// F:

	//sending deb files into containcer
	err = dockerClient.SendFileToContainer(ctx, sekaiDebFileName, debFileDestInContainer, SEKAID_CONTAINER_NAME)
	if err != nil {
		fmt.Println(err)
	}
	err = dockerClient.SendFileToContainer(ctx, interxDebFileName, debFileDestInContainer, INTERX_CONTAINER_NAME)
	if err != nil {
		fmt.Println(err)
	}
	err = dockerClient.InstallDebPackage(SEKAID_CONTAINER_NAME, debFileDestInContainer+sekaiDebFileName)
	if err != nil {
		fmt.Println(err)
	}
	err = dockerClient.InstallDebPackage(INTERX_CONTAINER_NAME, debFileDestInContainer+interxDebFileName)
	if err != nil {
		fmt.Println(err)
	}

	dockerClient.RunSekaidBin(ctx, SEKAID_CONTAINER_NAME)
	dockerClient.RunInterxBin(ctx, INTERX_CONTAINER_NAME, SEKAID_CONTAINER_NAME, strconv.Itoa(RPC_PORT), strconv.Itoa(GRPC_PORT))
	time.Sleep(time.Second * 10)
	os.Exit(1)

}
