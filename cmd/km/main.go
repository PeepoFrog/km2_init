package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/mrlutik/km2init/pkg/cosign"

	"github.com/Masterminds/semver"
)

const DockerImagePubKey = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE/IrzBQYeMwvKa44/DF/HB7XDpnE+
f+mU9F/Qbfq25bBWV2+NlYMJv3KvKHNtu3Jknt6yizZjUV4b8WGfKBzFYw==
-----END PUBLIC KEY-----`

func isValidSemVer(input string) error {
	_, err := semver.NewVersion(input)
	if err != nil {
		return err
	}
	return nil
}

func main() {

	var (
		baseImageVer    string
		baseImageName   string
		sekaiContainer  bool
		interxContainer bool
	)

	ctx := context.Background()
	// Set latest version of the base-image
	flag.StringVar(&baseImageVer, "image", "v0.13.7", "Base-image version. Default: v0.13.7")

	// Set contatiners to launch
	// Binary will be from master branch aka latest
	flag.BoolVar(&sekaiContainer, "sekai", false, "Set to true to start container with sekai")
	flag.BoolVar(&interxContainer, "interx", false, "Set to true to start container with interx")

	flag.Parse()

	// Define the image you want to pull
	if err := isValidSemVer(baseImageName); err != nil {
		fmt.Fprintln(os.Stderr, "semver is not valid")
		panic(err)
	}

	baseImageName = fmt.Sprintf("ghcr.io/kiracore/docker/base-image:%s", baseImageVer)

	err := cosign.VerifyDockerImage(ctx, baseImageName, DockerImagePubKey)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

}
