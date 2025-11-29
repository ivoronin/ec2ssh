package ssh

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSHKey(t *testing.T) {
	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	privateKeyPath, publicKey1, err := GenerateKeypair(tmpDir)
	require.NoError(t, err)
	assert.FileExists(t, privateKeyPath)
	assert.NotEmpty(t, privateKeyPath)

	publicKey2, err := GetPublicKey(privateKeyPath)
	require.NoError(t, err)
	assert.NotEmpty(t, publicKey2)

	assert.Equal(t, publicKey1, publicKey2)
}
