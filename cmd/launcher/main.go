package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/google/go-github/github"
	"github.com/mrlutik/km2_init/cmd/launcher/internal/adapters"
	"github.com/mrlutik/km2_init/cmd/launcher/internal/cosign"
	"github.com/mrlutik/km2_init/cmd/launcher/internal/docker"
	"github.com/mrlutik/km2_init/cmd/launcher/internal/helpers"
	"golang.org/x/oauth2"
)

const (
	NETWORK_NAME          = "testnet-1"
	SEKAID_HOME           = `/root/.sekaid-` + NETWORK_NAME
	KEYRING_BACKEND       = "test"
	DOCKER_IMAGE_NAME     = "ghcr.io/kiracore/docker/kira-base"
	DOCKER_IMAGE_VERSION  = "v0.13.11"
	DOCKER_NETWORK_NAME   = "kira_network"
	SEKAI_VERSION         = "latest" //or v0.3.16
	INTERX_VERSION        = "latest" //or v0.4.33
	SEKAID_CONTAINER_NAME = "sekaid"
	INTERX_CONTAINER_NAME = "interx"
	VOLUME_NAME           = "kira_volume:/data"
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

	launcherInterface := helpers.LauncherInterface(&helpers.Linux{})
	dockerBaseImageName := DOCKER_IMAGE_NAME + ":" + DOCKER_IMAGE_VERSION
	// dockerBaseImageName := "ubuntu"
	err := launcherInterface.PrivilageCheck()
	if err != nil {
		log.Fatalln(err)
	}

	arch, platform := launcherInterface.CheckPlaform()
	log.Println(arch, platform)
	ctx := context.Background()

	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		log.Fatalln(err)
	}
	defer dockerClient.Cli.Close()
	err = dockerClient.VerifyDockerInstallation(ctx)
	if err != nil {
		log.Fatalln(err)
		if err = launcherInterface.InstallDocker(); err != nil {
			log.Fatalln(err)
		}
		err = dockerClient.VerifyDockerInstallation(ctx)
		if err != nil {
			log.Fatalln(err)
		}
	}

	log.Println("docker installed")
	err = dockerClient.PullImage(ctx, dockerBaseImageName)
	if err != nil {
		log.Fatalln(err)
	}
	checkBool, err := cosign.VerifyImageSignature(ctx, dockerBaseImageName, DockerImagePubKey)
	if err != nil {
		log.Println(checkBool)
		log.Fatalln(err)
	}
	log.Println(checkBool)
	r := adapters.Repositories{}
	// kiraRepos := []string{"sekai", "interx"}
	sekaiRepo := "sekai"
	interxRepo := "interx"
	kiraGit := "KiraCore"
	r.Set(kiraGit, sekaiRepo, SEKAI_VERSION)
	r.Set(kiraGit, interxRepo, INTERX_VERSION)
	log.Println(r.Get())
	log.Println(os.LookupEnv("GITHUB_TOKEN"))
	token := os.Getenv("GITHUB_TOKEN")
	r = adapters.Fetch(r, token)
	log.Println(r)
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	githubClient := github.NewClient(tc)
	debFileDestInContainer := "/"
	sekaiDebFileName := "sekai-linux-amd64.deb"
	interxDebFileName := "interx-linux-amd64.deb"

	check, err := dockerClient.CheckForContainersName(ctx, SEKAID_CONTAINER_NAME)
	if err != nil {
		log.Fatalln(err)
	}
	if check {
		dockerClient.StopAndDeleteContainer(ctx, SEKAID_CONTAINER_NAME)
	}
	check, err = dockerClient.CheckForContainersName(ctx, INTERX_CONTAINER_NAME)
	if err != nil {
		log.Fatalln(err)
	}
	if check {
		dockerClient.StopAndDeleteContainer(ctx, INTERX_CONTAINER_NAME)
	}
	dockerClient.CheckAndCreateNetwork(ctx, DOCKER_NETWORK_NAME)

	sekaidManager, err := docker.NewSekaidManager(strconv.Itoa(GRPC_PORT), strconv.Itoa(RPC_PORT), dockerBaseImageName, VOLUME_NAME, DOCKER_NETWORK_NAME, dockerClient)
	if err != nil {
		log.Fatalln(err)
	}
	err = dockerClient.InitAndCreateContainer(ctx, sekaidManager.ContainerConfig, sekaidManager.SekaidNetworkingConfig, sekaidManager.SekaiHostConfig, SEKAID_CONTAINER_NAME)
	if err != nil {
		log.Fatalln(err)
	}

	interxManager, err := docker.NewInterxManager(strconv.Itoa(INTERX_PORT), dockerBaseImageName, VOLUME_NAME, DOCKER_NETWORK_NAME, dockerClient)
	if err != nil {
		log.Fatalln(err)
	}
	err = dockerClient.InitAndCreateContainer(ctx, interxManager.ContainerConfig, interxManager.SekaidNetworkingConfig, interxManager.SekaiHostConfig, INTERX_CONTAINER_NAME)
	if err != nil {
		log.Fatalln(err)
	}

	// goto F

	adapters.DownloadBinaryFromRepo(ctx, githubClient, kiraGit, sekaiRepo, sekaiDebFileName, SEKAI_VERSION)
	adapters.DownloadBinaryFromRepo(ctx, githubClient, kiraGit, interxRepo, interxDebFileName, INTERX_VERSION)
	// F:

	//sending deb files into containcer
	err = dockerClient.SendFileToContainer(ctx, sekaiDebFileName, debFileDestInContainer, SEKAID_CONTAINER_NAME)
	if err != nil {
		log.Fatalln("error while sending file to container", err)
	}
	err = dockerClient.SendFileToContainer(ctx, interxDebFileName, debFileDestInContainer, INTERX_CONTAINER_NAME)
	if err != nil {
		log.Fatalln(err)
	}
	err = dockerClient.InstallDebPackage(SEKAID_CONTAINER_NAME, debFileDestInContainer+sekaiDebFileName)
	if err != nil {
		log.Fatalln(err)
	}
	err = dockerClient.InstallDebPackage(INTERX_CONTAINER_NAME, debFileDestInContainer+interxDebFileName)
	if err != nil {
		log.Fatalln(err)
	}

	// err = dockerClient.RunSekaidBin(ctx, SEKAID_CONTAINER_NAME, NETWORK_NAME, SEKAID_HOME, KEYRING_BACKEND, strconv.Itoa(RPC_PORT))
	err = sekaidManager.RunSekaidContainer(SEKAID_CONTAINER_NAME, DOCKER_NETWORK_NAME, SEKAID_HOME, KEYRING_BACKEND, strconv.Itoa(RPC_PORT), MNEMONIC_FOLDER)
	if err != nil {
		log.Fatalln(err)
	}
	// err = dockerClient.RunInterxBin(ctx, INTERX_CONTAINER_NAME, SEKAID_CONTAINER_NAME, strconv.Itoa(RPC_PORT), strconv.Itoa(GRPC_PORT))
	err = interxManager.RunInterxContainer(SEKAID_CONTAINER_NAME, INTERX_CONTAINER_NAME, strconv.Itoa(RPC_PORT), strconv.Itoa(GRPC_PORT))
	if err != nil {
		log.Fatalln(err)
	}
	os.Exit(1)

}
