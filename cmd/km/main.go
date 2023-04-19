package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func main() {
	// Create a Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Unable to create Docker client: %s", err)
	}

	// Define the image you want to pull
	imageVer := "v0.13.7"
	imageName := fmt.Sprintf("ghcr.io/kiracore/docker/base-image:%s", imageVer)

	// Pull the Docker image
	err = pullImage(cli, imageName)
	if err != nil {
		log.Fatalf("Unable to pull Docker image: %s", err)
	}

	fmt.Printf("Successfully pulled image: %s\n", imageName)
}

func pullImage(cli *client.Client, imageName string) error {
	ctx := context.Background()

	options := types.ImagePullOptions{}
	reader, err := cli.ImagePull(ctx, imageName, options)
	if err != nil {
		return err
	}
	defer reader.Close()

	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return err
	}

	return nil
}
