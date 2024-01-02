package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
)

const sshKeyType = "ed25519"

func GenerateSSHKeypair(tmpDir string) (string, string, error) {
	privateKeyPath := path.Join(tmpDir, "id_"+sshKeyType)
	publicKeyPath := privateKeyPath + ".pub"
	cmd := exec.Command("ssh-keygen", "-q", "-t", sshKeyType, "-f", privateKeyPath, "-N", "")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("failed to generate keypair using ssh-keygen: %w", err)
	}

	publicKeyBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read generated public key file: %w", err)
	}

	publicKey := string(publicKeyBytes)

	return privateKeyPath, publicKey, nil
}

func GetSSHPublicKey(privateKeyPath string) (string, error) {
	cmd := exec.Command("ssh-keygen", "-y", "-f", privateKeyPath)

	outputBytes, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get public key using ssh-keygen: %w", err)
	}

	publicKey := string(outputBytes)

	return publicKey, nil
}
