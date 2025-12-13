package http

import (
	"net/http"
	wsDelivery "wetalk/internal/delivery/websocket"

	"github.com/go-chi/chi/v5"
)

func MapHttpRoutes(r *chi.Mux, httpHandler HttpHandler, websocketHandler wsDelivery.WebsocketHandler) {
	r.Handle("/ws/{userId}", http.HandlerFunc(websocketHandler.HandleWebSocket))

	// Chat routes
	r.Route("/chat", func(r chi.Router) {
		r.Post("/", http.HandlerFunc(httpHandler.CreateChat))
		r.Get("/{chatId}/messages", http.HandlerFunc(httpHandler.GetMessages))
	})

	// User routes
	r.Route("/user", func(r chi.Router) {
		r.Get("/{id}", http.HandlerFunc(httpHandler.GetUser))
		r.Get("/{id}/chat", http.HandlerFunc(httpHandler.ListChat))
	})
}
