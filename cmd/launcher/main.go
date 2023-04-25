package main

import (
	"context"
	"fmt"

	"github.com/mrlutik/km2_init/cmd/launcher/internal/cosign"
	"github.com/mrlutik/km2_init/cmd/launcher/internal/docker"
	"github.com/mrlutik/km2_init/cmd/launcher/internal/helpers"
	// "github.com/mrlutik/km2_init/cmd/launcher/internal/helpres"
	// "github.com/mrlutik/km2_init/cmd/launcher/internal/cosign"
	// "github.com/mrlutik/km2_init/cmd/launcher/internal/docker"
	// // "github.com/mrlutik/km2_init/cmd/launcher/internal/cosign"
	// "github.com/mrlutik/km2_init/cmd/launcher/internal/docker"
	// // "github.com/mrlutik/km2_init/internal/cosign"
	// "github.com/mrlutik/km2_init/internal/docker"
	// "github.com/mrlutik/km2_init/cmd/launcher/cosign"
	// "github.com/mrlutik/km2_init/cmd/launcher/docker"
	// "github.com/mrlutik/km2_init/cmd/launcher/helpres"
)

const (
	KIRA_BASE_VERSION = "0.13.7"
)
const DockerImagePubKey = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE/IrzBQYeMwvKa44/DF/HB7XDpnE+
f+mU9F/Qbfq25bBWV2+NlYMJv3KvKHNtu3Jknt6yizZjUV4b8WGfKBzFYw==
-----END PUBLIC KEY-----`

func main() {

	launcherInterface := helpers.LauncherInterface(&helpers.Linux{})
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

	b, err := cosign.VerifyImageSignature(ctx, dockerBaseImageName, DockerImagePubKey)
	if err != nil {
		fmt.Println(b)
		panic(err)
	}
	fmt.Println(b)
}
