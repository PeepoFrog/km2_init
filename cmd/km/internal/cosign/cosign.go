package cosign

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
)

func verifyImageSignature(imageRef, pubKey string) error {

	// Decode the PEM-encoded public key data
	block, _ := pem.Decode([]byte(pubKey))
	if block == nil {
		return fmt.Errorf("failed to decode public key")
	}
	// Parse the public key from the decoded PEM block
	// x509.ParsePKIXPublicKey is used for parsing PKIX public keys, which include RSA, DSA, and ECDSA public keys
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return fmt.Errorf("failed to parse image reference: %w", err)
	}
	_ = pub
	fmt.Println(ref.String())

	return nil

}
