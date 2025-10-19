package entity

type Chat struct {
	Id   string `bson:"_id" json:"id"`
	Name string `bson:"name" json:"name"`
}

type ChatParticipant struct {
	ChatId string `bson:"chatId" json:"chatId"`
	UserId string `bson:"userId" json:"userId"`
}
