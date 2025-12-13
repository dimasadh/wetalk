package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"wetalk/infrastructure/db"
	"wetalk/infrastructure/ws"
	httpHandler "wetalk/internal/delivery/http"
	"wetalk/internal/delivery/websocket"
	"wetalk/internal/repository"
	"wetalk/internal/usecase"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

func Run() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

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

 	// Check if Redis is enabled
    redisAddr := os.Getenv("REDIS_ADDR")
    useRedis := redisAddr != ""

    var hub ws.IHub
    if useRedis {
        serverID := os.Getenv("SERVER_ID")
        if serverID == "" {
            serverID = "server-1" // Default
        }

        log.Printf("Using Redis hub at %s with server ID: %s", redisAddr, serverID)
        redisHub := ws.NewRedisHub(redisAddr, serverID)
        hub = redisHub

        redisHub.SetOnClientUnregister(func(client *ws.UserClient) error {
            _, err := userUc.HandleUnregisterClient(ctx, client.UserId)
            return err
        })
    } else {
        log.Println("Using in-memory hub (single server)")
        memHub := ws.NewHub()
        hub = memHub

        memHub.SetOnClientUnregister(func(client *ws.UserClient) error {
            _, err := userUc.HandleUnregisterClient(ctx, client.UserId)
            return err
        })
    }

    go hub.Run()

	log.Println("Websocket is running")


	// CORS middleware
	router := chi.NewRouter()
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

	websocketH := websocket.NewWebsocketHandler(hub, userUc, messageUc, chatUc)
	httpH := httpHandler.NewHttpHandler(chatUc, userUc)
	httpHandler.MapHttpRoutes(router, *httpH, *websocketH)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("HTTP server is running on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}
