package wscat

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/gorilla/websocket"
)

type Websocket struct {
	conn *websocket.Conn
}

func NewWebSocket(uri string) (*Websocket, error) {
	conn, resp, err := websocket.DefaultDialer.Dial(uri, nil)
	if err != nil {
		if errors.Is(err, websocket.ErrBadHandshake) {
			return nil, fmt.Errorf("%w: %s", err, resp.Status)
		}
		return nil, err
	}

	return &Websocket{conn}, nil
}

func (w *Websocket) Close() {
	w.conn.Close()
}

func (w *Websocket) Reader() io.Reader {
	return &WebsocketReader{conn: w.conn}
}

func (w *Websocket) Writer() io.Writer {
	return &WebsocketWriter{w.conn}
}

type WebsocketReader struct {
	conn   *websocket.Conn
	buffer []byte
	mu     sync.Mutex // protects buffer
}

func (r *WebsocketReader) Read(buf []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.buffer) == 0 {
		_, msg, err := r.conn.ReadMessage()
		if err != nil {
			// Handle WebSocket close error
			var closeErr *websocket.CloseError
			if errors.As(err, &closeErr) {
				if closeErr.Code == websocket.CloseNormalClosure {
					return 0, io.EOF
				}
			}

			return 0, err
		}

		r.buffer = msg
	}

	n = copy(buf, r.buffer)
	r.buffer = r.buffer[n:]

	return n, nil
}

type WebsocketWriter struct {
	conn *websocket.Conn
}

func (w *WebsocketWriter) Write(buf []byte) (n int, err error) {
	err = w.conn.WriteMessage(websocket.BinaryMessage, buf)
	if err != nil {
		return 0, err
	}

	return len(buf), nil
}
