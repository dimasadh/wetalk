package usecase

import (
	"context"
	"wetalk/internal/entity"
	"wetalk/internal/repository"
)

type MessageUsecase interface {
	GetReceiver(ctx context.Context, chatId string) ([]string, error)
	SaveMessage(ctx context.Context, message entity.Message) (string, error)
	GetMessagesByChatId(ctx context.Context, chatId string, limit, offset int) ([]entity.Message, error)
	GetMessage(ctx context.Context, messageId string) (entity.Message, error)
	MarkAsRead(ctx context.Context, messageId string) error
}

type messageUsecase struct {
	messageRepo repository.MessageRepository
	chatRepo    repository.ChatRepository
	userRepo    repository.UserRepository
}

func NewMessageUseCase(messageRepo repository.MessageRepository, chatRepo repository.ChatRepository, userRepo repository.UserRepository) MessageUsecase {
	return &messageUsecase{
		messageRepo: messageRepo,
		chatRepo:    chatRepo,
		userRepo:    userRepo,
	}
}

func (m *messageUsecase) GetReceiver(ctx context.Context, chatId string) ([]string, error) {
	participants, err := m.chatRepo.GetParticipants(ctx, chatId)
	if err != nil {
		return nil, err
	}

	userIds := make([]string, 0, len(participants))
	for _, participant := range participants {
		userIds = append(userIds, participant.UserId)
	}

	return userIds, nil
}

func (m *messageUsecase) SaveMessage(ctx context.Context, message entity.Message) (string, error) {
	return m.messageRepo.Create(ctx, message)
}

func (m *messageUsecase) GetMessagesByChatId(ctx context.Context, chatId string, limit, offset int) ([]entity.Message, error) {
	return m.messageRepo.GetByChatId(ctx, chatId, limit, offset)
}

func (m *messageUsecase) GetMessage(ctx context.Context, messageId string) (entity.Message, error) {
	return m.messageRepo.Get(ctx, messageId)
}

func (m *messageUsecase) MarkAsRead(ctx context.Context, messageId string) error {
	message, err := m.messageRepo.Get(ctx, messageId)
	if err != nil {
		return err
	}

	message.IsRead = true
	return m.messageRepo.Update(ctx, message)
}