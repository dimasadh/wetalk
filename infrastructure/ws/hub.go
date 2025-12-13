package ws

import (
	"log"
	"sync"
)

type Hub struct {
	clients            map[string]*UserClient
	broadcast          chan []byte
	Register           chan *UserClient
	Unregister         chan *UserClient
	mu                 sync.RWMutex
	OnClientUnregister func(client *UserClient) error
}

func NewHub() IHub {
	return &Hub{
		clients:    make(map[string]*UserClient),
		broadcast:  make(chan []byte, 256),
		Register:   make(chan *UserClient),
		Unregister: make(chan *UserClient),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client.UserId] = client
			h.mu.Unlock()
			log.Printf("%s is connected", client.UserId)

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.UserId]; ok {
				delete(h.clients, client.UserId)
				close(client.send)
				log.Printf("%s is disconnected", client.UserId)
			}
			h.mu.Unlock()

			if h.OnClientUnregister != nil {
				if err := h.OnClientUnregister(client); err != nil {
					log.Printf("OnClientUnregister error: %v", err)
				}
			}

		case message := <-h.broadcast:
			h.mu.RLock()
			for userId, client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, userId)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}

func (h *Hub) SendToClient(clientID string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	client, exists := h.clients[clientID]
	if exists {
		select {
		case client.send <- message:
		default:
			log.Printf("Failed to send to client: %s", clientID)
		}
	}
}

func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) RegisterClient(client *UserClient) {
    h.Register <- client
}

func (h *Hub) UnregisterClient(client *UserClient) {
    h.Unregister <- client
}

func (h *Hub) SetOnClientUnregister(callback func(client *UserClient) error) {
    h.OnClientUnregister = callback
}
