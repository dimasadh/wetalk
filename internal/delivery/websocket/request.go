package websocket

type IncomingMessage struct {
	Message   string `json:"message"`
	ChatId    string `json:"chatId"`
	Timestamp int64  `json:"timestamp"`
}

type MessageReadAck struct {
	MessageId string `json:"messageId"`
	ChatId    string `json:"chatId"`
}
