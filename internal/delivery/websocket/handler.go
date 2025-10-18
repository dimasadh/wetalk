package websocket

import (
	"context"
	"log"
	"net/http"
	"strings"

	"wetalk/infrastructure/ws"
	"wetalk/internal/usecase"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Handler struct {
	hub       *ws.Hub
	userUc    usecase.UserUsecase
	messageUc usecase.MessageUsecase
}

func NewHandler(hub *ws.Hub, userUc usecase.UserUsecase, messageUc usecase.MessageUsecase) *Handler {
	return &Handler{
		hub:       hub,
		userUc:    userUc,
		messageUc: messageUc,
	}
}

func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleWebSocket called")
	ctx := r.Context()

	// get id from query param
	userId := strings.TrimPrefix(r.URL.Path, "/ws/")
	if userId == "" {
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	// find user
	user, err := h.userUc.Get(ctx, userId)
	log.Println("User fetched:", user)
	if err != nil {
		log.Printf("Get user error: %v", err)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}

	user.IsOnline = true
	err = h.userUc.Update(ctx, user)
	if err != nil {
		log.Printf("Update user error: %v", err)
		return
	}

	client := ws.NewClient(user.Id, h.hub, conn)
	h.hub.Register <- client

	go client.WritePump()
	client.ReadPump(func(data []byte) {
		h.handleMessage(client, data)
	})
}

func (h *Handler) HandleUnregisterClient(client *ws.UserClient) {
	ctx := context.Background()

	user, err := h.userUc.Get(ctx, client.UserId)
	if err != nil {
		log.Printf("Get user error: %v", err)
		return
	}

	user.IsOnline = false

	err = h.userUc.Update(ctx, user)
	if err != nil {
		log.Printf("HandleUnregisterClient error: %v", err)
		return
	}
}

func (h *Handler) handleMessage(client *ws.UserClient, data []byte) {
	msg := string(data)

	onlineUsers, err := h.userUc.GetOnlineUser(context.Background())
	if err != nil {
		log.Printf("GetOnlineUser error: %v", err)
		return
	}

	for _, user := range onlineUsers {
		if user.Id == client.UserId {
			continue
		}

		go h.hub.SendToClient(user.Id, []byte(msg))
	}
}
