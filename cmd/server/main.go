package server

import (
	"context"
	"log"
	"net/http"
	"wetalk/infrastructure/db"
	"wetalk/infrastructure/ws"
	"wetalk/internal/delivery/websocket"
	"wetalk/internal/repository"
	"wetalk/internal/usecase"
)

func Run() {
	ctx := context.Background()

	mongoDb, err := db.NewMongoStore(ctx, "mongodb://localhost:27017", "wetalk")
	if err != nil {
		panic(err)
	}

	log.Println("Connected to MongoDB")

	userRepo := repository.NewUserRepository(*mongoDb.DB)
	userUc := usecase.NewUserUseCase(userRepo)
	messageUc := usecase.NewMessageUseCase(userRepo)

	hub := ws.NewHub()
	websocketHandler := websocket.NewHandler(hub, userUc, messageUc)
	hub.OnClientUnregister = func(client *ws.UserClient) error {
		ctx := context.Background()

		_, err := userUc.HandleUnregisterClient(ctx, client.UserId)
		return err
	}

	go hub.Run()

	log.Println("Websocket is running")

	http.HandleFunc("/ws/", websocketHandler.HandleWebSocket)

	http.ListenAndServe(":8080", nil)
}
