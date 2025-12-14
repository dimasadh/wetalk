package usecase

import (
	"context"
	"errors"
	"fmt"

	"wetalk/internal/entity"
	"wetalk/internal/repository"
)

var (
	ErrChatNotFound          = errors.New("chat not found")
	ErrNotParticipant        = errors.New("you are not a participant of this chat")
	ErrNotAdmin              = errors.New("you are not an admin of this chat")
	ErrInvalidChatType       = errors.New("invalid chat type")
	ErrPersonalChatExists    = errors.New("personal chat with this user already exists")
	ErrCannotInviteToPersonal = errors.New("cannot invite users to personal chat")
	ErrAlreadyParticipant    = errors.New("user is already a participant")
	ErrInvitationNotFound    = errors.New("invitation not found")
	ErrInvalidInvitation     = errors.New("invalid invitation")
)

type ChatUsecase interface {
	// Chat operations
	Index(ctx context.Context, userId string) ([]entity.Chat, error)
	Get(ctx context.Context, chatId string, userId string) (entity.ChatDetailResponse, error)
	Delete(ctx context.Context, chatId string, userId string) error

	// Personal chat operations
	CreatePersonalChat(ctx context.Context, userId string, participantId string) (string, error)

	// Group chat operations
	CreateGroupChat(ctx context.Context, name string, description string, creatorId string, userIds []string) (string, error)
	InviteUsersToGroup(ctx context.Context, chatId string, inviterId string, userIds []string) error
	LeaveGroup(ctx context.Context, chatId string, userId string) error

	// Invitation operations
	GetPendingInvitations(ctx context.Context, userId string) ([]entity.ChatInvitation, error)
	RespondToInvitation(ctx context.Context, invitationId string, userId string, accept bool) error

	// Participant operations
	GetParticipants(ctx context.Context, chatId string, userId string) ([]entity.User, error)

	// Message operations
	GetMessages(ctx context.Context, chatId string, userId string, limit, offset int) ([]entity.Message, error)
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

// Index returns all chats that a user is participating in
func (c *chatUsecase) Index(ctx context.Context, userId string) ([]entity.Chat, error) {
	chats, err := c.chatRepo.Index(ctx, userId)
	if err != nil {
		return nil, err
	}

	// Collect all personal chat IDs
	var personalChatIds []string
	for _, chat := range chats {
		if chat.Type == entity.ChatTypePersonal {
			personalChatIds = append(personalChatIds, chat.Id)
		}
	}

	// If there are personal chats, fetch all their participants in bulk
	if len(personalChatIds) > 0 {
		participantsByChatId := make(map[string][]entity.ChatParticipant)

		for _, chatId := range personalChatIds {
			participants, err := c.chatRepo.GetParticipants(ctx, chatId)
			if err != nil {
				continue
			}
			participantsByChatId[chatId] = participants
		}

		userIdSet := make(map[string]bool)
		for _, participants := range participantsByChatId {
			for _, participant := range participants {
				if participant.UserId != userId {
					userIdSet[participant.UserId] = true
				}
			}
		}

		var userIds []string
		for uid := range userIdSet {
			userIds = append(userIds, uid)
		}

		var userMap map[string]entity.User
		if len(userIds) > 0 {
			userFilter := entity.UserIndexFilter{Ids: userIds}
			users, err := c.userRepo.Index(ctx, userFilter)
			if err == nil {
				userMap = make(map[string]entity.User)
				for _, user := range users {
					userMap[user.Id] = user
				}
			}
		}

		// Update chat names for personal chats
		if userMap != nil {
			for i, chat := range chats {
				if chat.Type == entity.ChatTypePersonal {
					participants, exists := participantsByChatId[chat.Id]
					if !exists {
						continue
					}

					// Find the other user
					for _, participant := range participants {
						if participant.UserId != userId {
							if otherUser, found := userMap[participant.UserId]; found {
								chats[i].Name = otherUser.Name
							}
							break
						}
					}
				}
			}
		}
	}

	return chats, nil
}

// Get returns a chat with its participants
func (c *chatUsecase) Get(ctx context.Context, chatId string, userId string) (entity.ChatDetailResponse, error) {
	isParticipant, err := c.chatRepo.IsParticipant(ctx, userId, chatId)
	if err != nil {
		return entity.ChatDetailResponse{}, err
	}
	if !isParticipant {
		return entity.ChatDetailResponse{}, ErrNotParticipant
	}

	chat, err := c.chatRepo.Get(ctx, chatId)
	if err != nil {
		return entity.ChatDetailResponse{}, err
	}

	participants, err := c.GetParticipants(ctx, chatId, userId)
	if err != nil {
		return entity.ChatDetailResponse{}, err
	}

	if chat.Type == entity.ChatTypePersonal {
		for _, participant := range participants {
			if participant.Id != userId {
				chat.Name = participant.Name
				break
			}
		}
	}

	return entity.ChatDetailResponse{
		Chat:         chat,
		Participants: participants,
	}, nil
}

// Delete deletes a chat (only creator/admin can delete)
func (c *chatUsecase) Delete(ctx context.Context, chatId string, userId string) error {
	// Get chat
	chat, err := c.chatRepo.Get(ctx, chatId)
	if err != nil {
		return err
	}

	if chat.CreatedBy != userId {
		isAdmin, err := c.chatRepo.IsAdmin(ctx, userId, chatId)
		if err != nil {
			return err
		}
		if !isAdmin {
			return ErrNotAdmin
		}
	}

	return c.chatRepo.Delete(ctx, chatId)
}

// CreatePersonalChat creates a 1-on-1 chat between two users
func (c *chatUsecase) CreatePersonalChat(ctx context.Context, userId string, participantId string) (string, error) {
	_, err := c.userRepo.Get(ctx, participantId)
	if err != nil {
		return "", fmt.Errorf("participant not found")
	}

	existingChat, err := c.chatRepo.GetPersonalChatBetweenUsers(ctx, userId, participantId)
	if err == nil {
		// Chat already exists, return its ID
		return existingChat.Id, nil
	}

	chat := entity.Chat{
		Name:      "Personal",
		Type:      entity.ChatTypePersonal,
		CreatedBy: userId,
	}

	chatId, err := c.chatRepo.Create(ctx, chat)
	if err != nil {
		return "", err
	}

	participants := []entity.ChatParticipant{
		{
			ChatId: chatId,
			UserId: userId,
			Role:   "member",
		},
		{
			ChatId: chatId,
			UserId: participantId,
			Role:   "member",
		},
	}

	err = c.chatRepo.AddParticipants(ctx, participants)
	if err != nil {
		return "", err
	}

	return chatId, nil
}

// CreateGroupChat creates a group chat with multiple users
func (c *chatUsecase) CreateGroupChat(ctx context.Context, name string, description string, creatorId string, userIds []string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("group name is required")
	}

	if len(userIds) == 0 {
		return "", fmt.Errorf("at least one participant is required")
	}

	userFilter := entity.UserIndexFilter{
		Ids: userIds,
	}
	users, err := c.userRepo.Index(ctx, userFilter)
	if err != nil {
		return "", err
	}

	if len(users) != len(userIds) {
		return "", fmt.Errorf("some user IDs are invalid")
	}

	chat := entity.Chat{
		Name:        name,
		Description: description,
		Type:        entity.ChatTypeGroup,
		CreatedBy:   creatorId,
	}

	chatId, err := c.chatRepo.Create(ctx, chat)
	if err != nil {
		return "", err
	}

	participants := []entity.ChatParticipant{
		{
			ChatId: chatId,
			UserId: creatorId,
			Role:   "admin",
		},
	}

	for _, userId := range userIds {
		if userId != creatorId {
			participants = append(participants, entity.ChatParticipant{
				ChatId: chatId,
				UserId: userId,
				Role:   "member",
			})
		}
	}

	err = c.chatRepo.AddParticipants(ctx, participants)
	if err != nil {
		return "", err
	}

	return chatId, nil
}

// InviteUsersToGroup invites users to a group chat
func (c *chatUsecase) InviteUsersToGroup(ctx context.Context, chatId string, inviterId string, userIds []string) error {
	chat, err := c.chatRepo.Get(ctx, chatId)
	if err != nil {
		return err
	}

	if chat.Type != entity.ChatTypeGroup {
		return ErrCannotInviteToPersonal
	}

	isParticipant, err := c.chatRepo.IsParticipant(ctx, inviterId, chatId)
	if err != nil {
		return err
	}
	if !isParticipant {
		return ErrNotParticipant
	}

	isAdmin, err := c.chatRepo.IsAdmin(ctx, inviterId, chatId)
	if err != nil {
		return err
	}
	if !isAdmin {
		return ErrNotAdmin
	}

	userFilter := entity.UserIndexFilter{
		Ids: userIds,
	}
	users, err := c.userRepo.Index(ctx, userFilter)
	if err != nil {
		return err
	}

	if len(users) != len(userIds) {
		return fmt.Errorf("some user IDs are invalid")
	}

	for _, userId := range userIds {
		isAlreadyParticipant, err := c.chatRepo.IsParticipant(ctx, userId, chatId)
		if err != nil {
			return err
		}
		if isAlreadyParticipant {
			continue // Skip if already a participant
		}

		_, err = c.chatRepo.GetInvitationByUserAndChat(ctx, userId, chatId)
		if err == nil {
			continue // Skip if invitation already exists
		}

		invitation := entity.ChatInvitation{
			ChatId:    chatId,
			InviterId: inviterId,
			InviteeId: userId,
		}

		_, err = c.chatRepo.CreateInvitation(ctx, invitation)
		if err != nil {
			return err
		}
	}

	return nil
}

// LeaveGroup allows a user to leave a group chat
func (c *chatUsecase) LeaveGroup(ctx context.Context, chatId string, userId string) error {
	chat, err := c.chatRepo.Get(ctx, chatId)
	if err != nil {
		return err
	}

	if chat.Type != entity.ChatTypeGroup {
		return fmt.Errorf("cannot leave personal chat")
	}

	isParticipant, err := c.chatRepo.IsParticipant(ctx, userId, chatId)
	if err != nil {
		return err
	}
	if !isParticipant {
		return ErrNotParticipant
	}

	return c.chatRepo.RemoveParticipant(ctx, userId, chatId)
}

// GetPendingInvitations returns all pending invitations for a user
func (c *chatUsecase) GetPendingInvitations(ctx context.Context, userId string) ([]entity.ChatInvitation, error) {
	return c.chatRepo.GetPendingInvitations(ctx, userId)
}

// RespondToInvitation allows a user to accept or reject an invitation
func (c *chatUsecase) RespondToInvitation(ctx context.Context, invitationId string, userId string, accept bool) error {
	invitation, err := c.chatRepo.GetInvitation(ctx, invitationId)
	if err != nil {
		return err
	}

	if invitation.InviteeId != userId {
		return ErrInvalidInvitation
	}

	if invitation.Status != "pending" {
		return fmt.Errorf("invitation has already been responded to")
	}

	status := "rejected"
	if accept {
		status = "accepted"
	}

	err = c.chatRepo.UpdateInvitationStatus(ctx, invitationId, status)
	if err != nil {
		return err
	}

	if accept {
		participants := []entity.ChatParticipant{
			{
				ChatId: invitation.ChatId,
				UserId: userId,
				Role:   "member",
			},
		}

		err = c.chatRepo.AddParticipants(ctx, participants)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetParticipants returns all participants of a chat
func (c *chatUsecase) GetParticipants(ctx context.Context, chatId string, userId string) ([]entity.User, error) {
	isParticipant, err := c.chatRepo.IsParticipant(ctx, userId, chatId)
	if err != nil {
		return nil, err
	}
	if !isParticipant {
		return nil, ErrNotParticipant
	}

	participants, err := c.chatRepo.GetParticipants(ctx, chatId)
	if err != nil {
		return nil, err
	}

	var userIds []string
	for _, participant := range participants {
		userIds = append(userIds, participant.UserId)
	}

	userFilter := entity.UserIndexFilter{
		Ids: userIds,
	}
	users, err := c.userRepo.Index(ctx, userFilter)
	if err != nil {
		return nil, err
	}

	for i := range users {
		users[i].Password = ""
	}

	return users, nil
}

// GetMessages returns messages for a chat
func (c *chatUsecase) GetMessages(ctx context.Context, chatId string, userId string, limit, offset int) ([]entity.Message, error) {
	isParticipant, err := c.chatRepo.IsParticipant(ctx, userId, chatId)
	if err != nil {
		return nil, err
	}
	if !isParticipant {
		return nil, ErrNotParticipant
	}

	return c.messageRepo.GetByChatId(ctx, chatId, limit, offset)
}
