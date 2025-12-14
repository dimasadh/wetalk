package entity

import "time"

type ChatType string

const (
	ChatTypePersonal ChatType = "personal"
	ChatTypeGroup    ChatType = "group"
)

type Chat struct {
	Id          string    `bson:"_id" json:"id"`
	Name        string    `bson:"name" json:"name"`
	Type        ChatType  `bson:"type" json:"type"`
	CreatedBy   string    `bson:"createdBy" json:"createdBy"`
	CreatedAt   time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time `bson:"updatedAt" json:"updatedAt"`
	Description string    `bson:"description,omitempty" json:"description,omitempty"`
}

type ChatParticipant struct {
	Id        string    `bson:"_id" json:"id"`
	ChatId    string    `bson:"chatId" json:"chatId"`
	UserId    string    `bson:"userId" json:"userId"`
	Role      string    `bson:"role" json:"role"` // "admin" or "member"
	JoinedAt  time.Time `bson:"joinedAt" json:"joinedAt"`
	IsActive  bool      `bson:"isActive" json:"isActive"`
}

type ChatInvitation struct {
	Id         string    `bson:"_id" json:"id"`
	ChatId     string    `bson:"chatId" json:"chatId"`
	InviterId  string    `bson:"inviterId" json:"inviterId"`
	InviteeId  string    `bson:"inviteeId" json:"inviteeId"`
	Status     string    `bson:"status" json:"status"` // "pending", "accepted", "rejected"
	CreatedAt  time.Time `bson:"createdAt" json:"createdAt"`
	RespondedAt *time.Time `bson:"respondedAt,omitempty" json:"respondedAt,omitempty"`
}

type ChatDetailResponse struct {
	Chat         Chat   `json:"chat"`
	Participants []User `json:"participants"`
}

type CreatePersonalChatRequest struct {
	ParticipantId string `json:"participantId"`
}

type CreateGroupChatRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	UserIds     []string `json:"userIds"`
}

type InviteUsersRequest struct {
	UserIds []string `json:"userIds"`
}

type RespondInvitationRequest struct {
	Accept bool `json:"accept"`
}
