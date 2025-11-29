package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"path"
)

const sshKeyType = "ed25519"

// GenerateKeypair generates an ed25519 SSH keypair in the given directory.
// Returns the private key path and the public key contents.
func GenerateKeypair(tmpDir string) (privateKeyPath, publicKey string, err error) {

	privateKeyPath = path.Join(tmpDir, "id_"+sshKeyType)
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

	return privateKeyPath, string(publicKeyBytes), nil
}

// GetPublicKey extracts the public key from an existing private key file.
func GetPublicKey(privateKeyPath string) (string, error) {
	cmd := exec.Command("ssh-keygen", "-y", "-f", privateKeyPath)

	outputBytes, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get public key using ssh-keygen: %w", err)
	}

	return string(outputBytes), nil
}
