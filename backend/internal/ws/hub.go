package ws

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/kurama/auction-system/backend/internal/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Hub struct {
	mu         sync.RWMutex
	rooms      map[string]map[*Client]bool // room -> clients
	register   chan *subscription
	unregister chan *subscription
	disconnect chan *Client // full disconnect (all rooms)
	rdb        *redis.Client
}

type subscription struct {
	room   string
	client *Client
}

func NewHub(rdb *redis.Client) *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *subscription, 256),
		unregister: make(chan *subscription, 256),
		disconnect: make(chan *Client, 256),
		rdb:        rdb,
	}
}

func (h *Hub) Run(ctx context.Context) {
	go h.subscribeRedis(ctx)

	for {
		select {
		case <-ctx.Done():
			h.closeAll()
			return

		case sub := <-h.register:
			h.mu.Lock()
			if h.rooms[sub.room] == nil {
				h.rooms[sub.room] = make(map[*Client]bool)
			}
			h.rooms[sub.room][sub.client] = true
			h.mu.Unlock()
			logger.Debug("client joined room", zap.String("room", sub.room))

		case sub := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.rooms[sub.room]; ok {
				delete(clients, sub.client)
				if len(clients) == 0 {
					delete(h.rooms, sub.room)
				}
			}
			h.mu.Unlock()
			logger.Debug("client left room", zap.String("room", sub.room))

		case client := <-h.disconnect:
			h.mu.Lock()
			for room, clients := range h.rooms {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					if len(clients) == 0 {
						delete(h.rooms, room)
					}
				}
			}
			h.mu.Unlock()
			close(client.send)
			logger.Debug("client disconnected from all rooms")
		}
	}
}

// BroadcastToRoom sends a message to all clients in a room via Redis pub/sub.
func (h *Hub) BroadcastToRoom(ctx context.Context, room string, msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error("failed to marshal broadcast message", zap.Error(err))
		return
	}

	payload := redisMessage{Room: room, Data: data}
	payloadBytes, _ := json.Marshal(payload)

	if err := h.rdb.Publish(ctx, "ws:broadcast", payloadBytes).Err(); err != nil {
		logger.Error("failed to publish to redis", zap.Error(err))
		h.localBroadcast(room, data)
	}
}

func (h *Hub) subscribeRedis(ctx context.Context) {
	sub := h.rdb.Subscribe(ctx, "ws:broadcast")
	defer sub.Close()

	ch := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			var rm redisMessage
			if err := json.Unmarshal([]byte(msg.Payload), &rm); err != nil {
				logger.Error("failed to unmarshal redis message", zap.Error(err))
				continue
			}
			h.localBroadcast(rm.Room, rm.Data)
		}
	}
}

func (h *Hub) localBroadcast(room string, data []byte) {
	h.mu.RLock()
	clients := h.rooms[room]
	h.mu.RUnlock()

	for client := range clients {
		select {
		case client.send <- data:
		default:
			logger.Warn("dropping message for slow client", zap.String("room", room))
		}
	}
}

func (h *Hub) Register(room string, client *Client) {
	h.register <- &subscription{room: room, client: client}
}

func (h *Hub) Unregister(room string, client *Client) {
	h.unregister <- &subscription{room: room, client: client}
}

// unregisterAll removes a client from all rooms and closes its send channel.
func (h *Hub) unregisterAll(client *Client) {
	h.disconnect <- client
}

func (h *Hub) RoomClientCount(room string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[room])
}

// closeAll closes all client connections on shutdown.
func (h *Hub) closeAll() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for room, clients := range h.rooms {
		for client := range clients {
			close(client.send)
			delete(clients, client)
		}
		delete(h.rooms, room)
	}
	logger.Info("all websocket connections closed")
}

type redisMessage struct {
	Room string          `json:"room"`
	Data json.RawMessage `json:"data"`
}
