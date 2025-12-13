package entity

type Message struct {
	Id        string `bson:"_id" json:"id"`
	ChatId    string `bson:"chatId" json:"chatId"`
	SenderId  string `bson:"senderId" json:"senderId"`
	Message   string `bson:"message" json:"message"`
	Timestamp int64  `bson:"timestamp" json:"timestamp"`
	IsRead    bool   `bson:"isRead" json:"isRead"`
}

type MessageIndexFilter struct {
	ChatId string `bson:"chatId"`
	Limit  int    `bson:"limit"`
	Offset int    `bson:"offset"`
}