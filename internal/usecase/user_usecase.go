package usecase

import (
	"context"
	"wetalk/internal/entity"
	"wetalk/internal/repository"
)

type UserUsecase interface {
	Get(ctx context.Context, userId string) (entity.User, error)
	Create(ctx context.Context, name string) (string, error)
	Update(ctx context.Context, user entity.User) error
	GetOnlineUser(ctx context.Context, userIds []string) ([]entity.User, error)
	HandleUnregisterClient(ctx context.Context, userId string) (string, error)
}

type userUsecase struct {
	userRepo repository.UserRepository
}

func NewUserUseCase(userRepo repository.UserRepository) UserUsecase {
	return &userUsecase{
		userRepo: userRepo,
	}
}

func (u *userUsecase) Get(ctx context.Context, userId string) (entity.User, error) {
	user, err := u.userRepo.Get(ctx, userId)
	if err != nil {
		return entity.User{}, err
	}

	return user, nil
}

func (u *userUsecase) Create(ctx context.Context, name string) (string, error) {
	user := entity.User{
		Name:     name,
		IsOnline: true,
	}

	return u.userRepo.Create(ctx, user)
}

func (u *userUsecase) Update(ctx context.Context, user entity.User) error {
	return u.userRepo.Update(ctx, user)
}

func (u *userUsecase) GetOnlineUser(ctx context.Context, userIds []string) ([]entity.User, error) {
	return u.userRepo.GetOnlineUser(ctx, userIds)
}

func (u *userUsecase) HandleUnregisterClient(ctx context.Context, userId string) (string, error) {
	user, err := u.userRepo.Get(ctx, userId)
	if err != nil {
		return "", err
	}

	user.IsOnline = false
	err = u.userRepo.Update(ctx, user)
	if err != nil {
		return "", err
	}

	return user.Id, nil
}
