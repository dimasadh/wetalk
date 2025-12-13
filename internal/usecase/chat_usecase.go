package usecase

import (
	"context"
	"fmt"

	"wetalk/internal/entity"
	"wetalk/internal/repository"
)

type ChatUsecase interface {
	Index(ctx context.Context, userId string) ([]entity.Chat, error)
	Get(ctx context.Context, chatId string) (entity.Chat, error)
	Create(ctx context.Context, name string, userIds []string) (string, error)
	AddParticipants(ctx context.Context, chatId string, userIds []string) error
	GetParticipants(ctx context.Context, chatId string) ([]entity.ChatParticipant, error)
	Delete(ctx context.Context, chatId string) error
	GetMessages(ctx context.Context, chatId string, limit, offset int) ([]entity.Message, error)
}

type chatUsecase struct {
	chatRepo    repository.ChatRepository
	userRepo    repository.UserRepository
	messageRepo repository.MessageRepository
}

func NewChatUsecase(chatRepo repository.ChatRepository, userRepo repository.UserRepository, messageRepo repository.MessageRepository) ChatUsecase {
	return &chatUsecase{
		chatRepo:    chatRepo,
		userRepo:    userRepo,
		messageRepo: messageRepo,
	}
}

func (c *chatUsecase) Index(ctx context.Context, userId string) ([]entity.Chat, error) {
	return c.chatRepo.Index(ctx, userId)
}

func (c *chatUsecase) Get(ctx context.Context, chatId string) (entity.Chat, error) {
	chat, err := c.chatRepo.Get(ctx, chatId)
	if err != nil {
		return entity.Chat{}, err
	}

	return chat, nil
}

func (c *chatUsecase) Create(ctx context.Context, name string, userIds []string) (string, error) {
	if len(userIds) == 0 {
		return "", fmt.Errorf("need at least one participant")
	}

	// validate userIds
	userFilter := entity.UserIndexFilter{
		Ids: userIds,
	}
	users, err := c.userRepo.Index(ctx, userFilter)
	if err != nil {
		return "", err
	}

	if len(users) != len(userIds) {
		return "", fmt.Errorf("some userIds are invalid")
	}

	chat := entity.Chat{
		Name: name,
	}

	chatId, err := c.chatRepo.Create(ctx, chat)
	if err != nil {
		return "", err
	}

	var participants []entity.ChatParticipant
	for _, userId := range userIds {
		participant := entity.ChatParticipant{
			ChatId: chatId,
			UserId: userId,
		}
		participants = append(participants, participant)
	}

	err = c.chatRepo.AddParticipants(ctx, participants)
	if err != nil {
		return "", err
	}

	return chatId, nil
}

func (c *chatUsecase) AddParticipants(ctx context.Context, chatId string, userIds []string) error {
	var participants []entity.ChatParticipant
	for _, userId := range userIds {
		participant := entity.ChatParticipant{
			ChatId: chatId,
			UserId: userId,
		}
		participants = append(participants, participant)
	}

	return c.chatRepo.AddParticipants(ctx, participants)
}

func (c *chatUsecase) GetParticipants(ctx context.Context, chatId string) ([]entity.ChatParticipant, error) {
	return c.chatRepo.GetParticipants(ctx, chatId)
}

func (c *chatUsecase) Delete(ctx context.Context, chatId string) error {
	return c.chatRepo.Delete(ctx, chatId)
}

func (c *chatUsecase) GetMessages(ctx context.Context, chatId string, limit, offset int) ([]entity.Message, error) {
	return c.messageRepo.GetByChatId(ctx, chatId, limit, offset)
}
