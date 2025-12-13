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
	httpHandler.MapHttpRoutes(router, *httpH, *websocketH)

	log.Println("HTTP server is running on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal(err)
	}
}
