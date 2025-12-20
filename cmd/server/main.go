package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"wetalk/infrastructure/db"
	"wetalk/infrastructure/ws"
	httpHandler "wetalk/internal/delivery/http"
	"wetalk/internal/delivery/websocket"
	"wetalk/internal/repository"
	"wetalk/internal/usecase"
	"wetalk/pkg/jwt"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

func Run() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("godotenv: error loading .env file")
	}

	ctx := context.Background()

	mongoDbHost := os.Getenv("MONGODB_URI")
	mongoDbName := os.Getenv("MONGODB_DATABASE")
	mongoDb, err := db.NewMongoStore(ctx, mongoDbHost, mongoDbName)
	if err != nil {
		panic(err)
	}

	log.Println("Connected to MongoDB")

	// Initialize repositories
	userRepo := repository.NewUserRepository(*mongoDb.DB)
	chatRepo := repository.NewChatRepository(*mongoDb.DB)
	messageRepo := repository.NewMessageRepository(*mongoDb.DB)
	refreshTokenRepo := repository.NewRefreshTokenRepository(*mongoDb.DB)

	// Initialize JWT manager
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-this-in-production" // Default for development
		log.Println("Warning: Using default JWT secret. Set JWT_SECRET in .env for production")
	}

	// Access token: 15 minutes, Refresh token: 30 days
	jwtManager := jwt.NewJWTManager(jwtSecret, 15*time.Minute, 30*24*time.Hour)

	// Initialize use cases
	authUc := usecase.NewAuthUsecase(userRepo, refreshTokenRepo, jwtManager)
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
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
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

	// Initialize handlers
	websocketH := websocket.NewWebsocketHandler(hub, userUc, messageUc, chatUc)
	httpH := httpHandler.NewHttpHandler(chatUc, userUc)
	authH := httpHandler.NewAuthHandler(authUc)
	authMiddleware := httpHandler.NewAuthMiddleware(authUc)

	// Map routes
	httpHandler.MapHttpRoutes(router, *httpH, *websocketH, *authH, authMiddleware)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("HTTP server is running on :%s", port)

	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}
