package tunnel

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockConnection implements TunnelConnection for testing.
type mockConnection struct {
	reader     io.Reader
	writer     io.Writer
	closeCount int
}

func (m *mockConnection) Reader() io.Reader { return m.reader }
func (m *mockConnection) Writer() io.Writer { return m.writer }
func (m *mockConnection) Close()            { m.closeCount++ }

// errorReader always returns an error.
type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (int, error) {
	return 0, e.err
}

// errorWriter always returns an error.
type errorWriter struct {
	err error
}

func (e *errorWriter) Write(p []byte) (int, error) {
	return 0, e.err
}

func TestRunWithIO_Success(t *testing.T) {
	t.Parallel()

	// Data to send through the tunnel
	inputData := "hello from stdin"
	connOutputData := "hello from connection"

	// Create mock connection with pre-loaded data
	connReader := strings.NewReader(connOutputData)
	connWriter := &bytes.Buffer{}
	conn := &mockConnection{
		reader: connReader,
		writer: connWriter,
	}

	// Mock dialer that returns our mock connection
	dialer := func(uri string) (TunnelConnection, error) {
		assert.Equal(t, "wss://test.uri", uri)
		return conn, nil
	}

	// Set up stdin and stdout
	stdin := strings.NewReader(inputData)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	// Run the tunnel
	err := RunWithIO("wss://test.uri", dialer, stdin, stdout, stderr)

	require.NoError(t, err)

	// Verify data flowed correctly
	assert.Equal(t, connOutputData, stdout.String(), "connection output should be written to stdout")
	assert.Equal(t, inputData, connWriter.String(), "stdin should be written to connection")
	assert.Equal(t, 1, conn.closeCount, "connection should be closed once")
	assert.Empty(t, stderr.String(), "no errors should be written to stderr")
}

func TestRunWithIO_DialError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("dial failed")

	dialer := func(uri string) (TunnelConnection, error) {
		return nil, expectedErr
	}

	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := RunWithIO("wss://test.uri", dialer, stdin, stdout, stderr)

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestRunWithIO_ReadError(t *testing.T) {
	t.Parallel()

	readErr := errors.New("read failed")
	conn := &mockConnection{
		reader: &errorReader{err: readErr},
		writer: io.Discard,
	}

	dialer := func(uri string) (TunnelConnection, error) {
		return conn, nil
	}

	// Use an empty stdin that will immediately EOF
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := RunWithIO("wss://test.uri", dialer, stdin, stdout, stderr)

	// Function completes without returning error (errors go to stderr)
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "read failed")
}

func TestRunWithIO_WriteError(t *testing.T) {
	t.Parallel()

	writeErr := errors.New("write failed")
	conn := &mockConnection{
		reader: strings.NewReader(""), // Will EOF immediately
		writer: &errorWriter{err: writeErr},
	}

	dialer := func(uri string) (TunnelConnection, error) {
		return conn, nil
	}

	stdin := strings.NewReader("data to write")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := RunWithIO("wss://test.uri", dialer, stdin, stdout, stderr)

	// Function completes without returning error (errors go to stderr)
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "write failed")
}

func TestRunWithIO_BinaryData(t *testing.T) {
	t.Parallel()

	// Test with binary data including null bytes
	binaryInput := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	binaryOutput := []byte{0xAA, 0xBB, 0xCC, 0x00, 0x11, 0x22}

	connWriter := &bytes.Buffer{}
	conn := &mockConnection{
		reader: bytes.NewReader(binaryOutput),
		writer: connWriter,
	}

	dialer := func(uri string) (TunnelConnection, error) {
		return conn, nil
	}

	stdin := bytes.NewReader(binaryInput)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := RunWithIO("wss://test.uri", dialer, stdin, stdout, stderr)

	require.NoError(t, err)
	assert.Equal(t, binaryOutput, stdout.Bytes(), "binary data should be preserved in output")
	assert.Equal(t, binaryInput, connWriter.Bytes(), "binary data should be preserved in input")
}

func TestRunWithIO_EmptyStreams(t *testing.T) {
	t.Parallel()

	conn := &mockConnection{
		reader: strings.NewReader(""),
		writer: io.Discard,
	}

	dialer := func(uri string) (TunnelConnection, error) {
		return conn, nil
	}

	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := RunWithIO("wss://test.uri", dialer, stdin, stdout, stderr)

	require.NoError(t, err)
	assert.Empty(t, stdout.String())
	assert.Empty(t, stderr.String())
}

func TestRunWithIO_LargeData(t *testing.T) {
	t.Parallel()

	// Test with data larger than typical buffer sizes
	largeData := bytes.Repeat([]byte("x"), 1024*1024) // 1MB

	connWriter := &bytes.Buffer{}
	conn := &mockConnection{
		reader: bytes.NewReader(largeData),
		writer: connWriter,
	}

	dialer := func(uri string) (TunnelConnection, error) {
		return conn, nil
	}

	stdin := bytes.NewReader(largeData)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := RunWithIO("wss://test.uri", dialer, stdin, stdout, stderr)

	require.NoError(t, err)
	assert.Equal(t, len(largeData), stdout.Len(), "large data should be fully transferred to stdout")
	assert.Equal(t, len(largeData), connWriter.Len(), "large data should be fully transferred to connection")
}

func TestDefaultDialer(t *testing.T) {
	t.Parallel()

	// Test that DefaultDialer returns an error for invalid URI
	// (can't test success without a real WebSocket server)
	_, err := DefaultDialer("wss://invalid.uri.that.does.not.exist.localhost:12345")

	require.Error(t, err)
}

func TestRun(t *testing.T) {
	t.Parallel()

	// Test that Run returns an error for invalid URI
	// This exercises the integration of Run -> RunWithIO -> DefaultDialer
	err := Run("wss://invalid.uri.that.does.not.exist.localhost:12345")

	require.Error(t, err)
}

// Ensure WebSocket implements TunnelConnection interface
var _ TunnelConnection = (*WebSocket)(nil)
