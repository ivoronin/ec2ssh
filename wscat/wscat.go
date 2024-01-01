package wscat

import (
	"fmt"
	"io"
	"os"
	"sync"
)

func pipe(src io.Reader, dst io.Writer, waitGroup *sync.WaitGroup) {
	if _, err := io.Copy(dst, src); err != nil {
		fmt.Fprintf(os.Stderr, "ec2ssh: error: %v\n", err)
	}

	waitGroup.Done()
}

func Run(uri string) error {
	webSocket, err := NewWebSocket(uri)
	if err != nil {
		return err
	}
	defer webSocket.Close()

	var waitGroup sync.WaitGroup

	waitGroup.Add(2)

	go pipe(webSocket.Reader(), os.Stdout, &waitGroup)
	go pipe(os.Stdin, webSocket.Writer(), &waitGroup)
	waitGroup.Wait()

	return nil
}
