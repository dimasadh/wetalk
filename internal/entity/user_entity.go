package entity

type User struct {
	Id       string `bson:"_id" json:"id"`
	Name     string `bson:"name" json:"name"`
	IsOnline bool   `bson:"isOnline" json:"isOnline"`
}

type UserIndexFilter struct {
	Ids []string `bson:"ids"`
}
