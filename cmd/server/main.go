package server

import (
	"context"
	"log"
	"net/http"
	"wetalk/infrastructure/db"
	"wetalk/infrastructure/ws"
	httpHandler "wetalk/internal/delivery/http"
	"wetalk/internal/delivery/websocket"
	"wetalk/internal/repository"
	"wetalk/internal/usecase"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func Run() {
	ctx := context.Background()

	mongoDb, err := db.NewMongoStore(ctx, "mongodb://localhost:27017", "wetalk")
	if err != nil {
		panic(err)
	}

	log.Println("Connected to MongoDB")

	userRepo := repository.NewUserRepository(*mongoDb.DB)
	chatRepo := repository.NewChatRepository(*mongoDb.DB)
	messageRepo := repository.NewMessageRepository(*mongoDb.DB)

	userUc := usecase.NewUserUseCase(userRepo)
	messageUc := usecase.NewMessageUseCase(messageRepo, chatRepo, userRepo)
	chatUc := usecase.NewChatUsecase(chatRepo, userRepo, messageRepo)

	httpH := httpHandler.NewHttpHandler(chatUc, userUc)

	hub := ws.NewHub()
	websocketH := websocket.NewWebsocketHandler(hub, userUc, messageUc, chatUc)
	hub.OnClientUnregister = func(client *ws.UserClient) error {
		ctx := context.Background()

		_, err := userUc.HandleUnregisterClient(ctx, client.UserId)
		return err
	}

	log.Println("Websocket is running")
	go hub.Run()

	router := chi.NewRouter()
	
	// CORS middleware
	router.Use(middleware.Logger)
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			
			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	})
	
	httpHandler.MapHttpRoutes(router, *httpH, *websocketH)

	log.Println("HTTP server is running on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal(err)
	}
}
