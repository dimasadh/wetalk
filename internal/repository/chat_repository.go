package repository

import (
	"context"
	"wetalk/internal/entity"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ChatRepository interface {
	Index(ctx context.Context, userId string) ([]entity.Chat, error)
	Get(ctx context.Context, chatId string) (entity.Chat, error)
	Create(ctx context.Context, chat entity.Chat) (string, error)
	AddParticipants(ctx context.Context, chatParticipants []entity.ChatParticipant) error
	GetParticipants(ctx context.Context, chatId string) ([]entity.ChatParticipant, error)
	Delete(ctx context.Context, chatId string) error
}

type chatRepository struct {
	db mongo.Database
}

func NewChatRepository(db mongo.Database) ChatRepository {
	return &chatRepository{
		db: db,
	}
}

func (r *chatRepository) Index(ctx context.Context, userId string) ([]entity.Chat, error) {
	collection := r.db.Collection("chats")

	matchStage := bson.D{{Key: "$match", Value: bson.D{{Key: "userId", Value: userId}}}}
	lookupStage := bson.D{{Key: "$lookup", Value: bson.D{
		{Key: "from", Value: "chat_participants"},
		{Key: "localField", Value: "_id"},
		{Key: "foreignField", Value: "chatId"},
		{Key: "as", Value: "participants"},
	}}}

	cursor, err := collection.Aggregate(ctx, mongo.Pipeline{matchStage, lookupStage})
	if err != nil {
		return nil, err
	}
	var chats []entity.Chat
	err = cursor.All(ctx, &chats)
	if err != nil {
		return nil, err
	}

	return chats, nil
}

func (r *chatRepository) Get(ctx context.Context, chatId string) (entity.Chat, error) {
	collection := r.db.Collection("chats")
	filter := bson.M{"_id": chatId}

	var chat entity.Chat
	err := collection.FindOne(ctx, filter).Decode(&chat)
	if err != nil {
		return entity.Chat{}, err
	}

	return chat, nil
}

func (r *chatRepository) Create(ctx context.Context, chat entity.Chat) (string, error) {
	collection := r.db.Collection("chats")

	_, err := collection.InsertOne(ctx, chat)
	if err != nil {
		return "", err
	}

	return chat.Id, nil
}

func (r *chatRepository) AddParticipants(ctx context.Context, chatParticipants []entity.ChatParticipant) error {
	collection := r.db.Collection("chat_participants")

	var participants []any
	for _, participant := range chatParticipants {
		participants = append(participants, participant)
	}

	_, err := collection.InsertMany(ctx, participants)
	if err != nil {
		return err
	}

	return nil
}

func (r *chatRepository) GetParticipants(ctx context.Context, chatId string) ([]entity.ChatParticipant, error) {
	collection := r.db.Collection("chat_participants")
	filter := bson.M{"chatId": chatId}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	var participants []entity.ChatParticipant
	err = cursor.All(ctx, &participants)
	if err != nil {
		return nil, err
	}

	return participants, nil
}

func (r *chatRepository) Delete(ctx context.Context, chatId string) error {
	collection := r.db.Collection("chats")
	filter := bson.M{"_id": chatId}
	_, err := collection.DeleteOne(ctx, filter)

	return err
}
