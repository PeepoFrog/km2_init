package cosign

import (
	"context"
	"fmt"
	"os"

	"github.com/sigstore/cosign/cmd/cosign/cli"
)

func VerifyDockerImage(ctx context.Context, image, pubKey string) error {
	co := &cli.VerifyCommandOptions{
		KeyRef: pubKey,
		RemoteOpts: cli.CommonRemoteOptions{
			Remote: "oci",
		},
	}
	verified, err := co.VerifyCmd(ctx, image)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	for _, vp := range verified {
		fmt.Fprintf(os.Stdout, "Verified signature for image '%s':\n", image)
		fmt.Fprintf(os.Stdout, "Payload: %s\n", string(vp.Payload))
		fmt.Fprintf(os.Stdout, "Cert: %s\n", string(vp.Cert))
	}
	return nil
}
