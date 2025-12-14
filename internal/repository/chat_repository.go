package repository

import (
	"context"
	"errors"
	"time"
	"wetalk/internal/entity"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ErrChatNotFound        = errors.New("chat not found")
	ErrNotParticipant      = errors.New("user is not a participant")
	ErrNotAdmin            = errors.New("user is not an admin")
	ErrInvitationNotFound  = errors.New("invitation not found")
	ErrPersonalChatExists  = errors.New("personal chat already exists")
)

type ChatRepository interface {
	// Chat operations
	Index(ctx context.Context, userId string) ([]entity.Chat, error)
	Get(ctx context.Context, chatId string) (entity.Chat, error)
	Create(ctx context.Context, chat entity.Chat) (string, error)
	Update(ctx context.Context, chat entity.Chat) error
	Delete(ctx context.Context, chatId string) error

	// Participant operations
	AddParticipants(ctx context.Context, chatParticipants []entity.ChatParticipant) error
	GetParticipants(ctx context.Context, chatId string) ([]entity.ChatParticipant, error)
	GetParticipantByUserAndChat(ctx context.Context, userId, chatId string) (entity.ChatParticipant, error)
	IsParticipant(ctx context.Context, userId, chatId string) (bool, error)
	IsAdmin(ctx context.Context, userId, chatId string) (bool, error)
	RemoveParticipant(ctx context.Context, userId, chatId string) error

	// Personal chat operations
	GetPersonalChatBetweenUsers(ctx context.Context, userId1, userId2 string) (entity.Chat, error)

	// Invitation operations
	CreateInvitation(ctx context.Context, invitation entity.ChatInvitation) (string, error)
	GetInvitation(ctx context.Context, invitationId string) (entity.ChatInvitation, error)
	GetPendingInvitations(ctx context.Context, userId string) ([]entity.ChatInvitation, error)
	UpdateInvitationStatus(ctx context.Context, invitationId, status string) error
	GetInvitationByUserAndChat(ctx context.Context, userId, chatId string) (entity.ChatInvitation, error)
}

type chatRepository struct {
	db mongo.Database
}

func NewChatRepository(db mongo.Database) ChatRepository {
	return &chatRepository{
		db: db,
	}
}

// Index returns all chats that a user is participating in
func (r *chatRepository) Index(ctx context.Context, userId string) ([]entity.Chat, error) {
	collection := r.db.Collection("chats")

	lookupStage := bson.D{{Key: "$lookup", Value: bson.D{
		{Key: "from", Value: "chat_participants"},
		{Key: "localField", Value: "_id"},
		{Key: "foreignField", Value: "chatId"},
		{Key: "as", Value: "participants"},
	}}}
	matchStage := bson.D{{Key: "$match", Value: bson.D{
		{Key: "participants.userId", Value: userId},
		{Key: "participants.isActive", Value: true},
	}}}
	sortStage := bson.D{{Key: "$sort", Value: bson.D{{Key: "updatedAt", Value: -1}}}}

	cursor, err := collection.Aggregate(ctx, mongo.Pipeline{lookupStage, matchStage, sortStage})
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

// Get returns a chat by ID
func (r *chatRepository) Get(ctx context.Context, chatId string) (entity.Chat, error) {
	collection := r.db.Collection("chats")
	filter := bson.M{"_id": chatId}

	var chat entity.Chat
	err := collection.FindOne(ctx, filter).Decode(&chat)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return entity.Chat{}, ErrChatNotFound
		}
		return entity.Chat{}, err
	}

	return chat, nil
}

// Create creates a new chat
func (r *chatRepository) Create(ctx context.Context, chat entity.Chat) (string, error) {
	collection := r.db.Collection("chats")
	chat.Id = uuid.New().String()
	chat.CreatedAt = time.Now()
	chat.UpdatedAt = time.Now()

	_, err := collection.InsertOne(ctx, chat)
	if err != nil {
		return "", err
	}

	return chat.Id, nil
}

// Update updates a chat
func (r *chatRepository) Update(ctx context.Context, chat entity.Chat) error {
	collection := r.db.Collection("chats")
	filter := bson.M{"_id": chat.Id}
	chat.UpdatedAt = time.Now()

	update := bson.M{
		"$set": bson.M{
			"name":        chat.Name,
			"description": chat.Description,
			"updatedAt":   chat.UpdatedAt,
		},
	}

	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}

// Delete deletes a chat
func (r *chatRepository) Delete(ctx context.Context, chatId string) error {
	collection := r.db.Collection("chats")
	filter := bson.M{"_id": chatId}
	_, err := collection.DeleteOne(ctx, filter)
	return err
}

// AddParticipants adds participants to a chat
func (r *chatRepository) AddParticipants(ctx context.Context, chatParticipants []entity.ChatParticipant) error {
	collection := r.db.Collection("chat_participants")

	var participants []interface{}
	for _, participant := range chatParticipants {
		participant.Id = uuid.New().String()
		participant.JoinedAt = time.Now()
		participant.IsActive = true
		participants = append(participants, participant)
	}

	_, err := collection.InsertMany(ctx, participants)
	if err != nil {
		return err
	}

	return nil
}

// GetParticipants returns all participants of a chat
func (r *chatRepository) GetParticipants(ctx context.Context, chatId string) ([]entity.ChatParticipant, error) {
	collection := r.db.Collection("chat_participants")
	filter := bson.M{
		"chatId":   chatId,
		"isActive": true,
	}

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

// GetParticipantByUserAndChat returns a specific participant
func (r *chatRepository) GetParticipantByUserAndChat(ctx context.Context, userId, chatId string) (entity.ChatParticipant, error) {
	collection := r.db.Collection("chat_participants")
	filter := bson.M{
		"userId":   userId,
		"chatId":   chatId,
		"isActive": true,
	}

	var participant entity.ChatParticipant
	err := collection.FindOne(ctx, filter).Decode(&participant)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return entity.ChatParticipant{}, ErrNotParticipant
		}
		return entity.ChatParticipant{}, err
	}

	return participant, nil
}

// IsParticipant checks if a user is a participant in a chat
func (r *chatRepository) IsParticipant(ctx context.Context, userId, chatId string) (bool, error) {
	collection := r.db.Collection("chat_participants")
	filter := bson.M{
		"userId":   userId,
		"chatId":   chatId,
		"isActive": true,
	}

	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// IsAdmin checks if a user is an admin of a chat
func (r *chatRepository) IsAdmin(ctx context.Context, userId, chatId string) (bool, error) {
	collection := r.db.Collection("chat_participants")
	filter := bson.M{
		"userId":   userId,
		"chatId":   chatId,
		"isActive": true,
		"role":     "admin",
	}

	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// RemoveParticipant removes a participant from a chat
func (r *chatRepository) RemoveParticipant(ctx context.Context, userId, chatId string) error {
	collection := r.db.Collection("chat_participants")
	filter := bson.M{
		"userId": userId,
		"chatId": chatId,
	}

	update := bson.M{
		"$set": bson.M{
			"isActive": false,
		},
	}

	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}

// GetPersonalChatBetweenUsers finds an existing personal chat between two users
func (r *chatRepository) GetPersonalChatBetweenUsers(ctx context.Context, userId1, userId2 string) (entity.Chat, error) {
	collection := r.db.Collection("chats")

	// Find chats where both users are participants and type is personal
	lookupStage := bson.D{{Key: "$lookup", Value: bson.D{
		{Key: "from", Value: "chat_participants"},
		{Key: "localField", Value: "_id"},
		{Key: "foreignField", Value: "chatId"},
		{Key: "as", Value: "participants"},
	}}}

	matchStage := bson.D{{Key: "$match", Value: bson.D{
		{Key: "type", Value: entity.ChatTypePersonal},
		{Key: "participants.userId", Value: bson.D{{Key: "$all", Value: bson.A{userId1, userId2}}}},
	}}}

	cursor, err := collection.Aggregate(ctx, mongo.Pipeline{lookupStage, matchStage})
	if err != nil {
		return entity.Chat{}, err
	}
	defer cursor.Close(ctx)

	var chats []entity.Chat
	if err := cursor.All(ctx, &chats); err != nil {
		return entity.Chat{}, err
	}

	if len(chats) == 0 {
		return entity.Chat{}, mongo.ErrNoDocuments
	}

	return chats[0], nil
}

// CreateInvitation creates a new chat invitation
func (r *chatRepository) CreateInvitation(ctx context.Context, invitation entity.ChatInvitation) (string, error) {
	collection := r.db.Collection("chat_invitations")

	invitation.Id = uuid.New().String()
	invitation.Status = "pending"
	invitation.CreatedAt = time.Now()

	_, err := collection.InsertOne(ctx, invitation)
	if err != nil {
		return "", err
	}

	return invitation.Id, nil
}

// GetInvitation returns an invitation by ID
func (r *chatRepository) GetInvitation(ctx context.Context, invitationId string) (entity.ChatInvitation, error) {
	collection := r.db.Collection("chat_invitations")
	filter := bson.M{"_id": invitationId}

	var invitation entity.ChatInvitation
	err := collection.FindOne(ctx, filter).Decode(&invitation)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return entity.ChatInvitation{}, ErrInvitationNotFound
		}
		return entity.ChatInvitation{}, err
	}

	return invitation, nil
}

// GetPendingInvitations returns all pending invitations for a user
func (r *chatRepository) GetPendingInvitations(ctx context.Context, userId string) ([]entity.ChatInvitation, error) {
	collection := r.db.Collection("chat_invitations")
	filter := bson.M{
		"inviteeId": userId,
		"status":    "pending",
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	var invitations []entity.ChatInvitation
	err = cursor.All(ctx, &invitations)
	if err != nil {
		return nil, err
	}

	return invitations, nil
}

// UpdateInvitationStatus updates the status of an invitation
func (r *chatRepository) UpdateInvitationStatus(ctx context.Context, invitationId, status string) error {
	collection := r.db.Collection("chat_invitations")
	filter := bson.M{"_id": invitationId}
	now := time.Now()

	update := bson.M{
		"$set": bson.M{
			"status":      status,
			"respondedAt": now,
		},
	}

	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}

// GetInvitationByUserAndChat finds a pending invitation for a user in a chat
func (r *chatRepository) GetInvitationByUserAndChat(ctx context.Context, userId, chatId string) (entity.ChatInvitation, error) {
	collection := r.db.Collection("chat_invitations")
	filter := bson.M{
		"inviteeId": userId,
		"chatId":    chatId,
		"status":    "pending",
	}

	var invitation entity.ChatInvitation
	err := collection.FindOne(ctx, filter).Decode(&invitation)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return entity.ChatInvitation{}, ErrInvitationNotFound
		}
		return entity.ChatInvitation{}, err
	}

	return invitation, nil
}
