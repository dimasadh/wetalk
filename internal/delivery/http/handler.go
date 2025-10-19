package http

import (
	"encoding/json"
	"log"
	"net/http"
	"wetalk/internal/usecase"
)

type HttpHandler struct {
	chatUc usecase.ChatUsecase
}

func NewHttpHandler(chatUc usecase.ChatUsecase) *HttpHandler {
	return &HttpHandler{
		chatUc: chatUc,
	}
}

type Response struct {
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// Method Post /chat
func (h *HttpHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
	r.Header.Set("Content-Type", "application/json")
	var req struct {
		Name    string   `json:"name"`
		UserIds []string `json:"userIds"`
	}

	var response Response
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Message = "invalid request body"
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	chatId, err := h.chatUc.Create(r.Context(), req.Name, req.UserIds)
	if err != nil {
		log.Printf("Create chat error: %v", err)
		response.Message = "internal server error"
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response.Message = "success"
	response.Data = map[string]string{"chatId": chatId}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Method Get user/:id/chat/
func (h *HttpHandler) ListChat(w http.ResponseWriter, r *http.Request) {
	userId := r.URL.Query().Get("id")

	response := Response{}
	chats, err := h.chatUc.Index(r.Context(), userId)
	if err != nil {
		log.Printf("List chat error: %v", err)
		response.Message = "internal server error"
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response.Message = "success"
	response.Data = chats
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
