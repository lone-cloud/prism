package subscription

type WebPushSubscription struct {
	Endpoint        string `json:"endpoint"`
	P256dh          string `json:"p256dh,omitempty"`
	Auth            string `json:"auth,omitempty"`
	VapidPrivateKey string `json:"vapidPrivateKey,omitempty"`
}

func (w *WebPushSubscription) HasEncryption() bool {
	return w.P256dh != "" && w.Auth != "" && w.VapidPrivateKey != ""
}

type SignalSubscription struct {
	GroupID string `json:"groupId"`
	Account string `json:"account"`
}

type TelegramSubscription struct {
	ChatID string `json:"chatId"`
}

type Subscription struct {
	ID       string                `json:"id"`
	AppName  string                `json:"appName"`
	Channel  Channel               `json:"channel"`
	Signal   *SignalSubscription   `json:"signal,omitempty"`
	WebPush  *WebPushSubscription  `json:"webPush,omitempty"`
	Telegram *TelegramSubscription `json:"telegram,omitempty"`
}

type App struct {
	AppName       string         `json:"appName"`
	Subscriptions []Subscription `json:"subscriptions"`
}
