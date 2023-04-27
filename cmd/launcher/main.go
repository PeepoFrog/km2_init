package main

import (
	"context"
	"fmt"

	"github.com/mrlutik/km2_init/cmd/launcher/internal/cosign"
	"github.com/mrlutik/km2_init/cmd/launcher/internal/docker"
	"github.com/mrlutik/km2_init/cmd/launcher/internal/helpers"
)

const (
	KIRA_BASE_VERSION = "0.13.7"
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
	dockerBaseImageName := "ghcr.io/kiracore/docker/kira-base:v" + KIRA_BASE_VERSION
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
}
