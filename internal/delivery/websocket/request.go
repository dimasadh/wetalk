package websocket

type IncomintMessage struct {
	Message   string `json:"message"`
	ChatId    string `json:"chatId"`
	Timestamp int64  `json:"timestamp"`
}
