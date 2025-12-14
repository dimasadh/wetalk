package entity

import "time"

type User struct {
	Id           string    `bson:"_id" json:"id"`
	Username     string    `bson:"username" json:"username"`
	Email        string    `bson:"email" json:"email"`
	Password     string    `bson:"password" json:"-"` // Don't expose password in JSON
	Name         string    `bson:"name" json:"name"`
	IsOnline     bool      `bson:"isOnline" json:"isOnline"`
	CreatedAt    time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time `bson:"updatedAt" json:"updatedAt"`
}

type UserIndexFilter struct {
	Ids []string `bson:"ids"`
}