package websocket

type OutgoingMessage struct {
	UserId    string `json:"chatId"`
	UserName  string `json:"userName"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}
