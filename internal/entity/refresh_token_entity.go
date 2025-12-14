package entity

import "time"

type RefreshToken struct {
	Id           string    `bson:"_id" json:"id"`
	UserId       string    `bson:"userId" json:"userId"`
	Token        string    `bson:"token" json:"token"`
	ExpiresAt    time.Time `bson:"expiresAt" json:"expiresAt"`
	CreatedAt    time.Time `bson:"createdAt" json:"createdAt"`
	RevokedAt    *time.Time `bson:"revokedAt,omitempty" json:"revokedAt,omitempty"`
	IsRevoked    bool      `bson:"isRevoked" json:"isRevoked"`
	DeviceInfo   string    `bson:"deviceInfo,omitempty" json:"deviceInfo,omitempty"`
	IpAddress    string    `bson:"ipAddress,omitempty" json:"ipAddress,omitempty"`
}