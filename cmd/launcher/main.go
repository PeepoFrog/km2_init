package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/github"
	"github.com/mrlutik/km2_init/cmd/launcher/internal/adapters"
	"github.com/mrlutik/km2_init/cmd/launcher/internal/cosign"
	"github.com/mrlutik/km2_init/cmd/launcher/internal/docker"
	"github.com/mrlutik/km2_init/cmd/launcher/internal/helpers"
	"golang.org/x/oauth2"
)

const (
	KIRA_BASE_VERSION = "v0.13.7"
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
	dockerBaseImageName := "ghcr.io/kiracore/docker/base-image:" + KIRA_BASE_VERSION
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
	sekaiVersion := "v0.3.14"
	interxVersion := "latest"
	r := adapters.Repositories{}
	kiraRepos := []string{"sekai", "interx"}
	kiraGit := "KiraCore"
	for _, v := range kiraRepos {
		switch v {
		case "sekai":
			r.Set(kiraGit, v, sekaiVersion)
		case "interx":
			r.Set(kiraGit, v, interxVersion)
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
	sekaidContainerName := "sekaiTest"
	interxContainerName := "interaxTest"
	// goto FINISH
	adapters.DownloadBinaryFromRepo(ctx, githubClient, "KiraCore", "sekai", "sekai-linux-amd64.deb", sekaiVersion)
	adapters.DownloadBinaryFromRepo(ctx, githubClient, "KiraCore", "interx", interxDebFileName, interxVersion)
	// FINISH:
	//sending deb files into containcer
	err = dockerClient.SendFileToContainer(ctx, sekaiDebFileName, debFileDestInContainer, sekaidContainerName)
	if err != nil {
		fmt.Println(err)
	}
	err = dockerClient.SendFileToContainer(ctx, interxDebFileName, debFileDestInContainer, interxContainerName)
	if err != nil {
		fmt.Println(err)
	}
	err = dockerClient.InstallDebPackage(sekaidContainerName, debFileDestInContainer+sekaiDebFileName)
	if err != nil {
		fmt.Println(err)
	}
	err = dockerClient.InstallDebPackage(interxContainerName, debFileDestInContainer+interxDebFileName)
	if err != nil {
		fmt.Println(err)
	}
	os.Exit(1)

}
