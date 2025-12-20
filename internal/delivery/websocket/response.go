package websocket

type OutgoingMessage struct {
	MessageId string `json:"messageId"`
	UserId    string `json:"userId"`
	UserName  string `json:"userName"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
	IsRead    bool   `json:"isRead"`
	ChatId    string `json:"chatId"`
}
