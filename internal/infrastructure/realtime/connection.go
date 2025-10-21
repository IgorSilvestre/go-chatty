package realtime

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pingPeriod = 30 * time.Second
)

// Connection wraps a websocket and coordinates outbound writes via a buffered channel.
// A connection is uniquely identified per user session and is safe for concurrent use.
type Connection struct {
	ID     string
	UserID string

	ws    *websocket.Conn
	send  chan []byte
	once  sync.Once
	close chan struct{}
}

// NewConnection constructs a Connection for the given user.
func NewConnection(userID string, ws *websocket.Conn) *Connection {
	return &Connection{
		ID:     uuid.NewString(),
		UserID: userID,
		ws:     ws,
		send:   make(chan []byte, 128),
		close:  make(chan struct{}),
	}
}

// Start launches the write loop. It must be called exactly once per connection.
func (c *Connection) Start() {
	go c.writeLoop()
}

// Send enqueues payload for delivery. If the client is slow and the buffer is full,
// the connection is closed to keep backpressure bounded.
func (c *Connection) Send(payload []byte) error {
	select {
	case <-c.close:
		return errors.New("connection closed")
	case c.send <- payload:
		return nil
	default:
		c.Close(websocket.CloseGoingAway, "send buffer full")
		return errors.New("connection buffer exceeded")
	}
}

// Close terminates the connection and stops the write loop.
func (c *Connection) Close(code int, reason string) {
	c.once.Do(func() {
		close(c.close)
		close(c.send)
		_ = c.ws.SetWriteDeadline(time.Now().Add(writeWait))
		_ = c.ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(writeWait))
		_ = c.ws.Close()
	})
}

func (c *Connection) writeLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-c.close:
			return
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			if err := c.writeMessage(msg); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.writePing(); err != nil {
				return
			}
		}
	}
}

func (c *Connection) writeMessage(payload []byte) error {
	if err := c.ws.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}
	return c.ws.WriteMessage(websocket.TextMessage, payload)
}

func (c *Connection) writePing() error {
	if err := c.ws.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}
	return c.ws.WriteMessage(websocket.PingMessage, nil)
}
