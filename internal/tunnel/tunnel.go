package tunnel

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// TunnelConnection represents a bidirectional tunnel connection.
// This interface enables testing with mock implementations.
type TunnelConnection interface {
	Reader() io.Reader
	Writer() io.Writer
	Close()
}

// Dialer is a function that creates a TunnelConnection from a URI.
type Dialer func(uri string) (TunnelConnection, error)

// DefaultDialer creates a WebSocket connection.
func DefaultDialer(uri string) (TunnelConnection, error) {
	return NewWebSocket(uri)
}

func pipe(src io.Reader, dst io.Writer, waitGroup *sync.WaitGroup, errCh chan<- error) {
	_, err := io.Copy(dst, src)
	if err != nil {
		errCh <- err
	}
	waitGroup.Done()
}

// RunWithIO starts a tunnel using the provided dialer and I/O streams.
// This function enables testing by allowing injection of mock connections and streams.
func RunWithIO(uri string, dial Dialer, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	conn, err := dial(uri)
	if err != nil {
		return err
	}
	defer conn.Close()

	var waitGroup sync.WaitGroup

	// Buffered channel to collect errors without blocking
	errCh := make(chan error, 2) //nolint:mnd

	waitGroup.Add(2) //nolint:mnd

	go pipe(conn.Reader(), stdout, &waitGroup, errCh)
	go pipe(stdin, conn.Writer(), &waitGroup, errCh)
	waitGroup.Wait()

	close(errCh)

	// Report first error if any occurred
	for err := range errCh {
		if err != nil {
			fmt.Fprintf(stderr, "ec2ssh: error: %v\n", err)
		}
	}

	return nil
}

// Run starts a WebSocket tunnel, piping stdin/stdout through the connection.
func Run(uri string) error {
	return RunWithIO(uri, DefaultDialer, os.Stdin, os.Stdout, os.Stderr)
}
