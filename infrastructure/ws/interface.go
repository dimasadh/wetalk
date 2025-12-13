package ws

type IHub interface {
    Run()
    RegisterClient(client *UserClient)
    UnregisterClient(client *UserClient)
    SendToClient(userID string, message []byte)
    Broadcast(message []byte)
    GetClientCount() int
    SetOnClientUnregister(callback func(client *UserClient) error)
}
