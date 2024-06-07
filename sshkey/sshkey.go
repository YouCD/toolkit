package sshkey

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"
)

func GenerateKey(bits int) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	private, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, fmt.Errorf("rsa.GenerateKey: %w", err)
	}
	return private, &private.PublicKey, nil
}

func EncodePrivateKey(private *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Bytes: x509.MarshalPKCS1PrivateKey(private),
		Type:  "RSA PRIVATE KEY",
	})
}

func EncodePublicKey(public *rsa.PublicKey) ([]byte, error) {
	publicBytes, err := x509.MarshalPKIXPublicKey(public)
	if err != nil {
		return nil, fmt.Errorf("marshalling public key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{
		Bytes: publicBytes,
		Type:  "PUBLIC KEY",
	}), nil
}

// EncodeSSHKey
func EncodeSSHKey(public *rsa.PublicKey) ([]byte, error) {
	publicKey, err := ssh.NewPublicKey(public)
	if err != nil {
		return nil, fmt.Errorf("ssh public key: %w", err)
	}
	return ssh.MarshalAuthorizedKey(publicKey), nil
}

func MakeSSHKeyPair() (string, string, error) {
	pkey, pubkey, err := GenerateKey(2048)
	if err != nil {
		return "", "", fmt.Errorf("generating ssh key: %w", err)
	}

	pub, err := EncodeSSHKey(pubkey)
	if err != nil {
		return "", "", fmt.Errorf("encoding ssh key: %w", err)
	}

	return string(EncodePrivateKey(pkey)), string(pub), nil
}
