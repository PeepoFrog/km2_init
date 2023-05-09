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

		r.Set(kiraGit, v)
		fmt.Println(r.Get())
	}

	fmt.Println(os.LookupEnv("GITHUB_TOKEN"))

	r = adapters.Fetch(r, os.Getenv("GITHUB_TOKEN"))
	fmt.Println(r)
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)
	githubClient := github.NewClient(tc)
	// goto FINISH
	adapters.DownloadBinaryFromRepo(ctx, githubClient, "KiraCore", "sekai", "sekai-linux-amd64.deb")
	adapters.DownloadBinaryFromRepo(ctx, githubClient, "KiraCore", "interx", "interx-linux-amd64.deb")
	// FINISH:
	// err = dockerClient.SendFileToDocker(ctx, "interaxTest", "/", "./interx-linux-amd64.deb")

	if err != nil {
		panic(err)
	}
	err = dockerClient.SendFileToContainer(ctx, "interx-linux-amd64.deb", "/", "interaxTest")
	if err != nil {
		fmt.Println(err)
	}
	os.Exit(1)

}
