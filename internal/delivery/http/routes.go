package http

import (
	"net/http"
	wsDelivery "wetalk/internal/delivery/websocket"

	"github.com/go-chi/chi/v5"
)

func MapHttpRoutes(r *chi.Mux, httpHandler HttpHandler, websocketHandler wsDelivery.WebsocketHandler, authHandler AuthHandler, authMiddleware *AuthMiddleware) {
	r.Handle("/ws/{userId}", http.HandlerFunc(websocketHandler.HandleWebSocket))

	// Auth routes (public)
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", http.HandlerFunc(authHandler.Register))
		r.Post("/login", http.HandlerFunc(authHandler.Login))
		r.Post("/refresh", http.HandlerFunc(authHandler.RefreshToken))
		r.Post("/logout", http.HandlerFunc(authHandler.Logout))
		
		// Protected auth routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Post("/logout-all", http.HandlerFunc(authHandler.LogoutAllDevices))
		})
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)

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
	})
}