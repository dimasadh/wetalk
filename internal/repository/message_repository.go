package repository

import (
	"context"
	"wetalk/internal/entity"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MessageRepository interface {
	Index(ctx context.Context, filter entity.MessageIndexFilter) ([]entity.Message, error)
	Get(ctx context.Context, messageId string) (entity.Message, error)
	Create(ctx context.Context, message entity.Message) (string, error)
	Update(ctx context.Context, message entity.Message) error
	Delete(ctx context.Context, messageId string) error
	GetByChatId(ctx context.Context, chatId string, limit, offset int) ([]entity.Message, error)
}

type messageRepository struct {
	db mongo.Database
}

func NewMessageRepository(db mongo.Database) MessageRepository {
	return &messageRepository{
		db: db,
	}
}

func (r *messageRepository) Index(ctx context.Context, filter entity.MessageIndexFilter) ([]entity.Message, error) {
	collection := r.db.Collection("messages")

	var bsonFilter bson.M
	if filter.ChatId != "" {
		bsonFilter = bson.M{"chatId": filter.ChatId}
	}

	opts := options.Find()
	if filter.Limit > 0 {
		opts.SetLimit(int64(filter.Limit))
	}
	if filter.Offset > 0 {
		opts.SetSkip(int64(filter.Offset))
	}
	opts.SetSort(bson.D{{Key: "timestamp", Value: -1}})

	cursor, err := collection.Find(ctx, bsonFilter, opts)
	if err != nil {
		return nil, err
	}

	var messages []entity.Message
	err = cursor.All(ctx, &messages)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

func (r *messageRepository) Get(ctx context.Context, messageId string) (entity.Message, error) {
	collection := r.db.Collection("messages")
	filter := bson.M{"_id": messageId}

	var message entity.Message
	err := collection.FindOne(ctx, filter).Decode(&message)
	if err != nil {
		return entity.Message{}, err
	}

	return message, nil
}

func (r *messageRepository) Create(ctx context.Context, message entity.Message) (string, error) {
	collection := r.db.Collection("messages")
	message.Id = uuid.New().String()

	_, err := collection.InsertOne(ctx, message)
	if err != nil {
		return "", err
	}

	return message.Id, nil
}

func (r *messageRepository) Update(ctx context.Context, message entity.Message) error {
	collection := r.db.Collection("messages")
	filter := bson.M{"_id": message.Id}
	update := bson.M{
		"$set": bson.M{
			"message":   message.Message,
			"isRead":    message.IsRead,
			"timestamp": message.Timestamp,
		},
	}
	_, err := collection.UpdateOne(ctx, filter, update)

	return err
}

func (r *messageRepository) Delete(ctx context.Context, messageId string) error {
	collection := r.db.Collection("messages")
	filter := bson.M{"_id": messageId}
	_, err := collection.DeleteOne(ctx, filter)

	return err
}

func (r *messageRepository) GetByChatId(ctx context.Context, chatId string, limit, offset int) ([]entity.Message, error) {
	collection := r.db.Collection("messages")
	filter := bson.M{"chatId": chatId}

	opts := options.Find()
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	if offset > 0 {
		opts.SetSkip(int64(offset))
	}
	opts.SetSort(bson.D{{Key: "timestamp", Value: -1}})

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	var messages []entity.Message
	err = cursor.All(ctx, &messages)
	if err != nil {
		return nil, err
	}

	return messages, nil
}