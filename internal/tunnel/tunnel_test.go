package tunnel

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockConnection implements TunnelConnection for testing
type mockConnection struct {
	reader *bytes.Buffer
	writer *bytes.Buffer
	closed bool
	mu     sync.Mutex
}

func newMockConnection() *mockConnection {
	return &mockConnection{
		reader: new(bytes.Buffer),
		writer: new(bytes.Buffer),
	}
}

func (m *mockConnection) Reader() io.Reader {
	return m.reader
}

func (m *mockConnection) Writer() io.Writer {
	return m.writer
}

func (m *mockConnection) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
}

func (m *mockConnection) isClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func TestRunWithIO_BidirectionalCopy(t *testing.T) {
	t.Parallel()

	// Setup mock connection with data to read
	conn := newMockConnection()
	conn.reader.WriteString("data from remote")

	// Setup local stdin data
	stdin := strings.NewReader("data from local")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	// Create a dialer that returns our mock
	dial := func(uri string) (TunnelConnection, error) {
		return conn, nil
	}

	// Run the tunnel
	err := RunWithIO("ws://test", dial, stdin, stdout, stderr)
	require.NoError(t, err)

	// Verify bidirectional copy
	assert.Equal(t, "data from remote", stdout.String(), "stdout should receive data from connection")
	assert.Equal(t, "data from local", conn.writer.String(), "connection should receive data from stdin")
	assert.True(t, conn.isClosed(), "connection should be closed")
	assert.Empty(t, stderr.String(), "no errors expected")
}

func TestRunWithIO_DialError(t *testing.T) {
	t.Parallel()

	dialErr := errors.New("connection refused")
	dial := func(uri string) (TunnelConnection, error) {
		return nil, dialErr
	}

	stdin := strings.NewReader("")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err := RunWithIO("ws://unreachable", dial, stdin, stdout, stderr)
	require.Error(t, err)
	assert.ErrorIs(t, err, dialErr)
}

func TestRunWithIO_URIPassed(t *testing.T) {
	t.Parallel()

	var receivedURI string
	conn := newMockConnection()

	dial := func(uri string) (TunnelConnection, error) {
		receivedURI = uri
		return conn, nil
	}

	stdin := strings.NewReader("")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err := RunWithIO("wss://example.com/tunnel?token=abc123", dial, stdin, stdout, stderr)
	require.NoError(t, err)
	assert.Equal(t, "wss://example.com/tunnel?token=abc123", receivedURI)
}

func TestRunWithIO_EmptyStreams(t *testing.T) {
	t.Parallel()

	conn := newMockConnection()
	dial := func(uri string) (TunnelConnection, error) {
		return conn, nil
	}

	stdin := strings.NewReader("")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err := RunWithIO("ws://test", dial, stdin, stdout, stderr)
	require.NoError(t, err)

	assert.Empty(t, stdout.String())
	assert.Empty(t, conn.writer.String())
	assert.True(t, conn.isClosed())
}

func TestRunWithIO_LargeData(t *testing.T) {
	t.Parallel()

	// Generate large data
	largeData := strings.Repeat("x", 64*1024) // 64KB

	conn := newMockConnection()
	conn.reader.WriteString(largeData)

	dial := func(uri string) (TunnelConnection, error) {
		return conn, nil
	}

	stdin := strings.NewReader(largeData)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err := RunWithIO("ws://test", dial, stdin, stdout, stderr)
	require.NoError(t, err)

	assert.Equal(t, len(largeData), stdout.Len(), "large data should be fully copied to stdout")
	assert.Equal(t, len(largeData), conn.writer.Len(), "large data should be fully copied to connection")
}

// errorReader simulates a reader that returns an error
type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

// errorWriter simulates a writer that returns an error
type errorWriter struct {
	err error
}

func (e *errorWriter) Write(p []byte) (n int, err error) {
	return 0, e.err
}

// errorConnection wraps mockConnection but returns error readers/writers
type errorConnection struct {
	*mockConnection
	readerErr error
	writerErr error
}

func (e *errorConnection) Reader() io.Reader {
	if e.readerErr != nil {
		return &errorReader{err: e.readerErr}
	}
	return e.mockConnection.Reader()
}

func (e *errorConnection) Writer() io.Writer {
	if e.writerErr != nil {
		return &errorWriter{err: e.writerErr}
	}
	return e.mockConnection.Writer()
}

func TestRunWithIO_ConnectionReadError(t *testing.T) {
	t.Parallel()

	readErr := errors.New("connection reset")
	conn := &errorConnection{
		mockConnection: newMockConnection(),
		readerErr:      readErr,
	}

	dial := func(uri string) (TunnelConnection, error) {
		return conn, nil
	}

	stdin := strings.NewReader("")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	// RunWithIO doesn't return the copy error, it prints to stderr
	err := RunWithIO("ws://test", dial, stdin, stdout, stderr)
	require.NoError(t, err) // RunWithIO returns nil even on copy errors

	// Error should be written to stderr
	assert.Contains(t, stderr.String(), "connection reset")
}

func TestRunWithIO_ConnectionWriteError(t *testing.T) {
	t.Parallel()

	writeErr := errors.New("broken pipe")
	conn := &errorConnection{
		mockConnection: newMockConnection(),
		writerErr:      writeErr,
	}

	dial := func(uri string) (TunnelConnection, error) {
		return conn, nil
	}

	stdin := strings.NewReader("data to send")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err := RunWithIO("ws://test", dial, stdin, stdout, stderr)
	require.NoError(t, err) // RunWithIO returns nil even on copy errors

	// Error should be written to stderr
	assert.Contains(t, stderr.String(), "broken pipe")
}
