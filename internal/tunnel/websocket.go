package tunnel

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/gorilla/websocket"
)

// WebSocket wraps a gorilla websocket connection with io.Reader/Writer interfaces.
type WebSocket struct {
	conn *websocket.Conn
}

// NewWebSocket dials the given URI and returns a WebSocket wrapper.
func NewWebSocket(uri string) (*WebSocket, error) {
	conn, resp, err := websocket.DefaultDialer.Dial(uri, nil)
	if err != nil {
		if errors.Is(err, websocket.ErrBadHandshake) {
			return nil, fmt.Errorf("%w: %s", err, resp.Status)
		}

		return nil, err
	}

	return &WebSocket{conn}, nil
}

// Close closes the underlying WebSocket connection.
func (w *WebSocket) Close() {
	_ = w.conn.Close()
}

// Reader returns an io.Reader that reads from the WebSocket.
func (w *WebSocket) Reader() io.Reader {
	return &websocketReader{conn: w.conn}
}

// Writer returns an io.Writer that writes to the WebSocket.
func (w *WebSocket) Writer() io.Writer {
	return &websocketWriter{w.conn}
}

type websocketReader struct {
	conn   *websocket.Conn
	buffer []byte
	mu     sync.Mutex // protects buffer
}

func (r *websocketReader) Read(buf []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.buffer) == 0 {
		_, msg, err := r.conn.ReadMessage()
		if err != nil {
			// Handle WebSocket close error
			var closeErr *websocket.CloseError
			if errors.As(err, &closeErr) && closeErr.Code == websocket.CloseNormalClosure {
				return 0, io.EOF
			}

			return 0, err
		}

		r.buffer = msg
	}

	n := copy(buf, r.buffer)
	r.buffer = r.buffer[n:]

	return n, nil
}

type websocketWriter struct {
	conn *websocket.Conn
}

func (w *websocketWriter) Write(buf []byte) (int, error) {
	err := w.conn.WriteMessage(websocket.BinaryMessage, buf)
	if err != nil {
		return 0, err
	}

	return len(buf), nil
}
