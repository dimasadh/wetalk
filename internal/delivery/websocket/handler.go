package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"wetalk/infrastructure/ws"
	"wetalk/internal/entity"
	"wetalk/internal/usecase"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebsocketHandler struct {
	hub       *ws.Hub
	userUc    usecase.UserUsecase
	messageUc usecase.MessageUsecase
	chatUc    usecase.ChatUsecase
}

func NewWebsocketHandler(hub *ws.Hub, userUc usecase.UserUsecase, messageUc usecase.MessageUsecase, chatUc usecase.ChatUsecase) *WebsocketHandler {
	return &WebsocketHandler{
		hub:       hub,
		userUc:    userUc,
		messageUc: messageUc,
		chatUc:    chatUc,
	}
}

func (h *WebsocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userId := chi.URLParam(r, "userId")
	if userId == "" {
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	user, err := h.userUc.Get(ctx, userId)
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
		h.handleMessage(ctx, client, data)
	})
}

func (h *WebsocketHandler) HandleUnregisterClient(client *ws.UserClient) {
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

func (h *WebsocketHandler) handleMessage(ctx context.Context, client *ws.UserClient, data []byte) {
	var message IncomintMessage
	err := json.Unmarshal(data, &message)
	if err != nil {
		log.Printf("Unknown message: %v", err)
		return
	}

	chat, err := h.chatUc.Get(ctx, message.ChatId)
	if err != nil {
		log.Printf("Get chat error: %v", err)
		return
	}

	sender, err := h.userUc.Get(ctx, client.UserId)
	if err != nil {
		log.Printf("Get sender user error: %v", err)
		return
	}

	participants, err := h.chatUc.GetParticipants(ctx, chat.Id)
	if err != nil {
		log.Printf("GetParticipants error: %v", err)
		return
	}

	if len(participants) == 0 {
		log.Printf("No participants in chat: %s", chat.Id)
		h.chatUc.Delete(ctx, chat.Id)
		return
	}

	userIds := make([]string, len(participants))
	for _, participant := range participants {
		userIds = append(userIds, participant.UserId)
	}

	var offlineUsers []entity.User
	onlineUsers, err := h.userUc.GetOnlineUser(ctx, userIds)
	if err != nil {
		log.Printf("GetOnlineUser error: %v", err)
		return
	}

	userMap := make(map[string]bool)
	for _, user := range onlineUsers {
		userMap[user.Id] = true
	}

	var wg sync.WaitGroup

	for _, participant := range participants {
		if participant.UserId == client.UserId {
			continue
		}
		wg.Add(1)
		go func(userId string) {
			defer wg.Done()
			if _, exists := userMap[userId]; !exists {
				return
			}

			message := OutgoingMessage{
				UserId:    client.UserId,
				UserName:  sender.Name,
				Message:   message.Message,
				Timestamp: message.Timestamp,
			}
			messageBytes, err := json.Marshal(message)
			if err != nil {
				log.Printf("Marshal message error: %v", err)
				return
			}

			h.hub.SendToClient(userId, messageBytes)

		}(participant.UserId)
	}

	wg.Wait()

	// Save message to inbox if recipient is offline
	for _, userId := range userIds {
		if _, exists := userMap[userId]; !exists {
			offlineUsers = append(offlineUsers, entity.User{Id: userId})
		}
	}

	if len(offlineUsers) > 0 {
		// implement later
	}
}
