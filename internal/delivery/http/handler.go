package http

import (
	"encoding/json"
	"log"
	"net/http"
	"wetalk/internal/entity"
	"wetalk/internal/usecase"

	"github.com/go-chi/chi/v5"
)

type HttpHandler struct {
	chatUc usecase.ChatUsecase
	userUc usecase.UserUsecase
}

func NewHttpHandler(chatUc usecase.ChatUsecase, userUc usecase.UserUsecase) *HttpHandler {
	return &HttpHandler{
		chatUc: chatUc,
		userUc: userUc,
	}
}

type Response struct {
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// GET /user/chats - Get list of chats for authenticated user
func (h *HttpHandler) ListUserChats(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	userClaims, ok := r.Context().Value(UserContextKey).(*entity.TokenClaims)
	if !ok {
		response := Response{Message: "unauthorized"}
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	chats, err := h.chatUc.Index(r.Context(), userClaims.UserId)
	if err != nil {
		log.Printf("List chats error: %v", err)
		response := Response{Message: "internal server error"}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{
		Message: "success",
		Data:    chats,
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// POST /chat/personal - Create a personal chat (1-on-1)
func (h *HttpHandler) CreatePersonalChat(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	userClaims, ok := r.Context().Value(UserContextKey).(*entity.TokenClaims)
	if !ok {
		response := Response{Message: "unauthorized"}
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	var req entity.CreatePersonalChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := Response{Message: "invalid request body"}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	if req.ParticipantId == "" {
		response := Response{Message: "participantId is required"}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	if req.ParticipantId == userClaims.UserId {
		response := Response{Message: "cannot create chat with yourself"}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	chatId, err := h.chatUc.CreatePersonalChat(r.Context(), userClaims.UserId, req.ParticipantId)
	if err != nil {
		log.Printf("Create personal chat error: %v", err)
		response := Response{Message: "failed to create personal chat"}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{
		Message: "personal chat created successfully",
		Data:    map[string]string{"chatId": chatId},
	}
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// POST /chat/group - Create a group chat
func (h *HttpHandler) CreateGroupChat(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	userClaims, ok := r.Context().Value(UserContextKey).(*entity.TokenClaims)
	if !ok {
		response := Response{Message: "unauthorized"}
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	var req entity.CreateGroupChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := Response{Message: "invalid request body"}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	if req.Name == "" {
		response := Response{Message: "group name is required"}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	if len(req.UserIds) == 0 {
		response := Response{Message: "at least one participant is required"}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	chatId, err := h.chatUc.CreateGroupChat(r.Context(), req.Name, req.Description, userClaims.UserId, req.UserIds)
	if err != nil {
		log.Printf("Create group chat error: %v", err)
		response := Response{Message: "failed to create group chat"}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{
		Message: "group chat created successfully",
		Data:    map[string]string{"chatId": chatId},
	}
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GET /chat/:chatId - Get chat details with participants
func (h *HttpHandler) GetChat(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	userClaims, ok := r.Context().Value(UserContextKey).(*entity.TokenClaims)
	if !ok {
		response := Response{Message: "unauthorized"}
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	chatId := chi.URLParam(r, "chatId")
	if chatId == "" {
		response := Response{Message: "chatId is required"}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	chatDetail, err := h.chatUc.Get(r.Context(), chatId, userClaims.UserId)
	if err != nil {
		log.Printf("Get chat error: %v", err)

		statusCode := http.StatusInternalServerError
		message := "internal server error"

		switch err {
			case usecase.ErrNotParticipant:
				statusCode = http.StatusForbidden
				message = "you are not a participant of this chat"
			case usecase.ErrChatNotFound:
				statusCode = http.StatusNotFound
				message = "chat not found"
		}

		response := Response{Message: message}
		w.WriteHeader(statusCode)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{
		Message: "success",
		Data:    chatDetail,
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GET /chat/:chatId/messages - Get messages for a chat
func (h *HttpHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	userClaims, ok := r.Context().Value(UserContextKey).(*entity.TokenClaims)
	if !ok {
		response := Response{Message: "unauthorized"}
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	chatId := chi.URLParam(r, "chatId")
	if chatId == "" {
		response := Response{Message: "chatId is required"}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	messages, err := h.chatUc.GetMessages(r.Context(), chatId, userClaims.UserId, 100, 0)
	if err != nil {
		log.Printf("Get messages error: %v", err)

		statusCode := http.StatusInternalServerError
		message := "internal server error"

		if err == usecase.ErrNotParticipant {
			statusCode = http.StatusForbidden
			message = "you are not a participant of this chat"
		}

		response := Response{Message: message}
		w.WriteHeader(statusCode)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{
		Message: "success",
		Data:    messages,
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// POST /chat/:chatId/invite - Invite users to a group chat
func (h *HttpHandler) InviteUsersToGroup(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	userClaims, ok := r.Context().Value(UserContextKey).(*entity.TokenClaims)
	if !ok {
		response := Response{Message: "unauthorized"}
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	chatId := chi.URLParam(r, "chatId")
	if chatId == "" {
		response := Response{Message: "chatId is required"}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	var req entity.InviteUsersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := Response{Message: "invalid request body"}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	if len(req.UserIds) == 0 {
		response := Response{Message: "at least one user is required"}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	err := h.chatUc.InviteUsersToGroup(r.Context(), chatId, userClaims.UserId, req.UserIds)
	if err != nil {
		log.Printf("Invite users error: %v", err)

		statusCode := http.StatusInternalServerError
		message := "failed to invite users"

		if err == usecase.ErrNotParticipant {
			statusCode = http.StatusForbidden
			message = "you are not a participant of this chat"
		} else if err == usecase.ErrNotAdmin {
			statusCode = http.StatusForbidden
			message = "only admins can invite users"
		} else if err == usecase.ErrCannotInviteToPersonal {
			statusCode = http.StatusBadRequest
			message = "cannot invite users to personal chat"
		}

		response := Response{Message: message}
		w.WriteHeader(statusCode)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{
		Message: "invitations sent successfully",
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// POST /chat/:chatId/leave - Leave a group chat
func (h *HttpHandler) LeaveGroup(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	userClaims, ok := r.Context().Value(UserContextKey).(*entity.TokenClaims)
	if !ok {
		response := Response{Message: "unauthorized"}
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	chatId := chi.URLParam(r, "chatId")
	if chatId == "" {
		response := Response{Message: "chatId is required"}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	err := h.chatUc.LeaveGroup(r.Context(), chatId, userClaims.UserId)
	if err != nil {
		log.Printf("Leave group error: %v", err)

		statusCode := http.StatusInternalServerError
		message := "failed to leave group"

		if err == usecase.ErrNotParticipant {
			statusCode = http.StatusForbidden
			message = "you are not a participant of this chat"
		}

		response := Response{Message: message}
		w.WriteHeader(statusCode)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{
		Message: "left group successfully",
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GET /invitations - Get pending invitations for authenticated user
func (h *HttpHandler) GetPendingInvitations(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	userClaims, ok := r.Context().Value(UserContextKey).(*entity.TokenClaims)
	if !ok {
		response := Response{Message: "unauthorized"}
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	invitations, err := h.chatUc.GetPendingInvitations(r.Context(), userClaims.UserId)
	if err != nil {
		log.Printf("Get invitations error: %v", err)
		response := Response{Message: "internal server error"}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{
		Message: "success",
		Data:    invitations,
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// POST /invitations/:invitationId/respond - Accept or reject an invitation
func (h *HttpHandler) RespondToInvitation(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	userClaims, ok := r.Context().Value(UserContextKey).(*entity.TokenClaims)
	if !ok {
		response := Response{Message: "unauthorized"}
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	invitationId := chi.URLParam(r, "invitationId")
	if invitationId == "" {
		response := Response{Message: "invitationId is required"}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	var req entity.RespondInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := Response{Message: "invalid request body"}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	err := h.chatUc.RespondToInvitation(r.Context(), invitationId, userClaims.UserId, req.Accept)
	if err != nil {
		log.Printf("Respond to invitation error: %v", err)

		statusCode := http.StatusInternalServerError
		message := "failed to respond to invitation"

		if err == usecase.ErrInvitationNotFound {
			statusCode = http.StatusNotFound
			message = "invitation not found"
		} else if err == usecase.ErrInvalidInvitation {
			statusCode = http.StatusForbidden
			message = "invalid invitation"
		}

		response := Response{Message: message}
		w.WriteHeader(statusCode)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	message := "invitation rejected"
	if req.Accept {
		message = "invitation accepted"
	}

	response := Response{
		Message: message,
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GET /user/:id - Get user by ID
func (h *HttpHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	userId := chi.URLParam(r, "id")

	response := Response{}
	user, err := h.userUc.Get(r.Context(), userId)
	if err != nil {
		log.Printf("Get user error: %v", err)
		response.Message = "user not found"
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response.Message = "success"
	response.Data = user
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DELETE /chat/:chatId - Delete a chat (admin only)
func (h *HttpHandler) DeleteChat(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	userClaims, ok := r.Context().Value(UserContextKey).(*entity.TokenClaims)
	if !ok {
		response := Response{Message: "unauthorized"}
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	chatId := chi.URLParam(r, "chatId")
	if chatId == "" {
		response := Response{Message: "chatId is required"}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	err := h.chatUc.Delete(r.Context(), chatId, userClaims.UserId)
	if err != nil {
		log.Printf("Delete chat error: %v", err)

		statusCode := http.StatusInternalServerError
		message := "failed to delete chat"

		if err == usecase.ErrNotAdmin {
			statusCode = http.StatusForbidden
			message = "only admins can delete the chat"
		} else if err == usecase.ErrChatNotFound {
			statusCode = http.StatusNotFound
			message = "chat not found"
		}

		response := Response{Message: message}
		w.WriteHeader(statusCode)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{
		Message: "chat deleted successfully",
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
