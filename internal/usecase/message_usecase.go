package usecase

import "wetalk/internal/repository"

type MessageUsecase interface {
	GetReceiver(userId, message string) ([]string, error)
}

type messageUsecase struct {
	UserRepo repository.UserRepository
}

func NewMessageUseCase(userRepository repository.UserRepository) MessageUsecase {
	return &messageUsecase{
		UserRepo: userRepository,
	}
}

func (m *messageUsecase) GetReceiver(userId, message string) ([]string, error) {
	return []string{}, nil
}
