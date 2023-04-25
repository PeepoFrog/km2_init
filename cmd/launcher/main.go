package main

import (
	"context"
	"fmt"

	"github.com/mrlutik/km2_init/internal/cosign"
	"github.com/mrlutik/km2_init/internal/docker"
	"github.com/mrlutik/km2_init/internal/launcher"
)

const (
	KIRA_BASE_VERSION = "0.13.7"
)

func main() {
	launcherInterface := launcher.LauncherInterface(&launcher.Linux{})
	dockerBaseImageName := "ghcr.io/kiracore/docker/kira-base:v" + KIRA_BASE_VERSION
	err := launcherInterface.PrivilageCheck()
	if err != nil {
		panic(err)
	}
	arch, platform := launcherInterface.CheckPlaform()
	fmt.Println(arch, platform)
	ctx := context.Background()
	err = docker.PullImage(ctx, dockerBaseImageName)
	if err != nil {
		panic(err)
	}
	fmt.Println("test")

	b, err := cosign.VerifyImageSignature(ctx, dockerBaseImageName, cosign.DockerImagePubKey)
	if err != nil {
		fmt.Println(b)
		panic(err)
	}
	fmt.Println(b)
}
