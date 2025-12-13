// wetalk/infrastructure/ws/hub_redis.go
package ws

import (
    "context"
    "encoding/json"
    "log"
    "sync"

    "github.com/redis/go-redis/v9"
)

type RedisHub struct {
    // Local connections (in-memory map)
    clients    map[string]*UserClient
    mu         sync.RWMutex

    // Redis for distributed messaging
    redisClient *redis.Client
    pubsub      *redis.PubSub
    serverID    string

    // Channels
    Register   chan *UserClient
    Unregister chan *UserClient
    broadcast  chan []byte

    // Callbacks
    OnClientUnregister func(client *UserClient) error
}

type RedisMessage struct {
    FromServerID string `json:"fromServerId"`
    ToUserID     string `json:"toUserId"`
    Payload      []byte `json:"payload"`
}

func NewRedisHub(redisAddr string, serverID string) IHub {
    rdb := redis.NewClient(&redis.Options{
        Addr: redisAddr,
    })

    hub := &RedisHub{
        clients:     make(map[string]*UserClient),
        redisClient: rdb,
        serverID:    serverID,
        Register:    make(chan *UserClient),
        Unregister:  make(chan *UserClient),
        broadcast:   make(chan []byte, 256),
    }

    // Subscribe to Redis channels
    hub.pubsub = rdb.PSubscribe(context.Background(), "messages:*")

    return hub
}

func (h *RedisHub) Run() {
    // Start Redis subscriber in separate goroutine
    go h.subscribeRedis()

    for {
        select {
        case client := <-h.Register:
            h.mu.Lock()
            h.clients[client.UserId] = client
            h.mu.Unlock()

            // Announce this user is on this server
            h.redisClient.Set(
                context.Background(),
                "user:"+client.UserId+":server",
                h.serverID,
                0, // No expiration (or use TTL with heartbeat)
            )

            log.Printf("[%s] %s connected", h.serverID, client.UserId)

        case client := <-h.Unregister:
            h.mu.Lock()
            if _, ok := h.clients[client.UserId]; ok {
                delete(h.clients, client.UserId)
                close(client.send)

                // Remove from Redis
                h.redisClient.Del(
                    context.Background(),
                    "user:"+client.UserId+":server",
                )

                log.Printf("[%s] %s disconnected", h.serverID, client.UserId)
            }
            h.mu.Unlock()

            if h.OnClientUnregister != nil {
                if err := h.OnClientUnregister(client); err != nil {
                    log.Printf("OnClientUnregister error: %v", err)
                }
            }

        case message := <-h.broadcast:
            h.broadcastLocal(message)
        }
    }
}

// Subscribe to Redis messages (CONSUMER)
func (h *RedisHub) subscribeRedis() {
    ch := h.pubsub.Channel()

    log.Printf("[%s] Redis subscriber started", h.serverID)

    for msg := range ch {
        // Received message from Redis
        var redisMsg RedisMessage
        if err := json.Unmarshal([]byte(msg.Payload), &redisMsg); err != nil {
            log.Printf("Error unmarshaling Redis message: %v", err)
            continue
        }

        // Don't process messages we sent ourselves
        if redisMsg.FromServerID == h.serverID {
            continue
        }

        h.mu.RLock()
        _, existsLocally := h.clients[redisMsg.ToUserID]
        h.mu.RUnlock()
        if !existsLocally {
      		continue
        }


        log.Printf("[%s] Received message from Redis for user %s",
            h.serverID, redisMsg.ToUserID)

        // Send to local client if connected here
        h.SendToClient(redisMsg.ToUserID, redisMsg.Payload)
    }
}

// Send to specific client (checks local first, then Redis)
func (h *RedisHub) SendToClient(userID string, message []byte) {
    h.mu.RLock()
    client, existsLocally := h.clients[userID]
    h.mu.RUnlock()

    if existsLocally {
        // Fast path: User is connected to THIS server
        select {
        case client.send <- message:
            log.Printf("[%s] Sent message to local client %s", h.serverID, userID)
        default:
            log.Printf("[%s] Failed to send to local client %s", h.serverID, userID)
        }
    } else {
        // Slow path: User might be on ANOTHER server
        // Publish to Redis for other servers to handle
        h.publishToRedis(userID, message)
    }
}

// Publish to Redis (PRODUCER)
func (h *RedisHub) publishToRedis(userID string, message []byte) {
    ctx := context.Background()

    redisMsg := RedisMessage{
        FromServerID: h.serverID,
        ToUserID:     userID,
        Payload:      message,
    }

    msgBytes, err := json.Marshal(redisMsg)
    if err != nil {
        log.Printf("Error marshaling Redis message: %v", err)
        return
    }

    // Publish to specific user channel
    err = h.redisClient.Publish(ctx, "messages:"+userID, msgBytes).Err()
    if err != nil {
        log.Printf("Error publishing to Redis: %v", err)
        return
    }

    log.Printf("[%s] Published message to Redis for user %s", h.serverID, userID)
}

// Broadcast to all local clients
func (h *RedisHub) broadcastLocal(message []byte) {
    h.mu.RLock()
    defer h.mu.RUnlock()

    for userId, client := range h.clients {
        select {
        case client.send <- message:
        default:
            log.Printf("Failed to send to client: %s", userId)
        }
    }
}

func (h *RedisHub) Broadcast(message []byte) {
    h.broadcast <- message
}

func (h *RedisHub) GetClientCount() int {
    h.mu.RLock()
    defer h.mu.RUnlock()
    return len(h.clients)
}

func (h *RedisHub) RegisterClient(client *UserClient) {
    h.Register <- client
}

func (h *RedisHub) UnregisterClient(client *UserClient) {
    h.Unregister <- client
}

func (h *RedisHub) SetOnClientUnregister(callback func(client *UserClient) error) {
    h.OnClientUnregister = callback
}
