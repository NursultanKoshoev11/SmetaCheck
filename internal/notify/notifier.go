package notify

type Message struct{
 UserID string
 Title string
 Body string
 Link string
}

type Notifier interface{
 Send(Message) error
}
