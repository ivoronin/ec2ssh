package ssh

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests require real ssh-keygen to be installed

func TestGenerateKeypair(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	privateKeyPath, publicKey, err := GenerateKeypair(tmpDir)
	require.NoError(t, err)

	// Check private key file was created
	assert.Equal(t, filepath.Join(tmpDir, "id_ed25519"), privateKeyPath)
	assert.FileExists(t, privateKeyPath)

	// Check private key has correct permissions (read/write only by owner)
	info, err := os.Stat(privateKeyPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "private key should have 0600 permissions")

	// Check public key file was created
	publicKeyPath := privateKeyPath + ".pub"
	assert.FileExists(t, publicKeyPath)

	// Check public key has correct format
	assert.True(t, strings.HasPrefix(publicKey, "ssh-ed25519 "), "public key should start with ssh-ed25519")
	assert.True(t, len(publicKey) > 50, "public key should have substantial length")

	// Verify the public key content matches the file
	fileContent, err := os.ReadFile(publicKeyPath)
	require.NoError(t, err)
	assert.Equal(t, publicKey, string(fileContent))
}

func TestGenerateKeypair_InvalidDirectory(t *testing.T) {
	t.Parallel()

	// Try to generate keypair in non-existent directory
	_, _, err := GenerateKeypair("/nonexistent/path/that/does/not/exist")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate keypair")
}

func TestGenerateKeypair_MultipleGenerations(t *testing.T) {
	t.Parallel()

	// Generate multiple keypairs in different directories
	// to verify independence and uniqueness

	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	path1, pub1, err := GenerateKeypair(tmpDir1)
	require.NoError(t, err)

	path2, pub2, err := GenerateKeypair(tmpDir2)
	require.NoError(t, err)

	// Different directories should produce different paths
	assert.NotEqual(t, path1, path2)

	// Keys should be unique
	assert.NotEqual(t, pub1, pub2)
}

func TestGetPublicKey(t *testing.T) {
	t.Parallel()

	// First generate a keypair
	tmpDir := t.TempDir()
	privateKeyPath, expectedPublicKey, err := GenerateKeypair(tmpDir)
	require.NoError(t, err)

	// Now extract the public key using GetPublicKey
	publicKey, err := GetPublicKey(privateKeyPath)
	require.NoError(t, err)

	// Should match the key from generation
	assert.Equal(t, expectedPublicKey, publicKey)
}

func TestGetPublicKey_InvalidPath(t *testing.T) {
	t.Parallel()

	_, err := GetPublicKey("/nonexistent/key/path")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get public key")
}

func TestGetPublicKey_InvalidKeyFile(t *testing.T) {
	t.Parallel()

	// Create a file that's not a valid SSH key
	tmpDir := t.TempDir()
	invalidKeyPath := filepath.Join(tmpDir, "not_a_key")
	err := os.WriteFile(invalidKeyPath, []byte("this is not a valid ssh key"), 0o600)
	require.NoError(t, err)

	_, err = GetPublicKey(invalidKeyPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get public key")
}

func TestGenerateKeypair_KeyType(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	privateKeyPath, _, err := GenerateKeypair(tmpDir)
	require.NoError(t, err)

	// Read the private key file to verify it's ed25519
	content, err := os.ReadFile(privateKeyPath)
	require.NoError(t, err)

	// Ed25519 keys start with this header
	assert.Contains(t, string(content), "OPENSSH PRIVATE KEY")
}
