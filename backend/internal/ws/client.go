package ws

import (
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024
	sendBufSize    = 256
)

// ClientMessage is a message sent from client to server.
type ClientMessage struct {
	Action string `json:"action"` // "subscribe", "unsubscribe"
	Room   string `json:"room"`   // e.g. "auction:uuid", "user:uuid"
}

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	rooms  map[string]bool
	userID string // authenticated user ID (empty if anonymous)
}

func NewClient(hub *Hub, conn *websocket.Conn, userID string) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, sendBufSize),
		rooms:  make(map[string]bool),
		userID: userID,
	}
}

// canSubscribe checks if the client is allowed to subscribe to a room.
// Public rooms (auction:*) are open to all.
// Private rooms (user:*) require matching userID.
func (c *Client) canSubscribe(room string) bool {
	parts := strings.SplitN(room, ":", 2)
	if len(parts) != 2 {
		return false
	}

	switch parts[0] {
	case "auction":
		return true // Public — anyone can watch
	case "user":
		return c.userID != "" && parts[1] == c.userID // Private — only own room
	default:
		return false
	}
}

// ReadPump reads messages from the websocket connection.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregisterAll(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Error("websocket read error", "error", err)
			}
			break
		}

		var msg ClientMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		switch msg.Action {
		case "subscribe":
			if msg.Room != "" && !c.rooms[msg.Room] && c.canSubscribe(msg.Room) {
				c.rooms[msg.Room] = true
				c.hub.Register(msg.Room, c)
			}
		case "unsubscribe":
			if c.rooms[msg.Room] {
				delete(c.rooms, msg.Room)
				c.hub.Unregister(msg.Room, c)
			}
		}
	}
}

// WritePump pumps messages from the hub to the websocket connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
