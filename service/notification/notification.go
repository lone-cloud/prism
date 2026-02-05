package notification

type Action struct {
	ID       string                 `json:"id"`
	Endpoint string                 `json:"endpoint"`
	Method   string                 `json:"method"`
	Data     map[string]interface{} `json:"data,omitempty"`
}

type Notification struct {
	Title   string   `json:"title,omitempty"`
	Message string   `json:"message"`
	Actions []Action `json:"actions,omitempty"`
}

type Channel string

const (
	ChannelSignal   Channel = "signal"
	ChannelWebPush  Channel = "webpush"
	ChannelTelegram Channel = "telegram"
)

func (c Channel) IsAvailable(signalEnabled bool, telegramEnabled bool) bool {
	switch c {
	case ChannelWebPush:
		return true
	case ChannelSignal:
		return signalEnabled
	case ChannelTelegram:
		return telegramEnabled
	default:
		return false
	}
}

type WebPushSubscription struct {
	Endpoint        string
	P256dh          string
	Auth            string
	VapidPrivateKey string
}

func (w *WebPushSubscription) HasEncryption() bool {
	return w.P256dh != "" && w.Auth != "" && w.VapidPrivateKey != ""
}

type SignalSubscription struct {
	GroupID string
	Account string
}

type Mapping struct {
	AppName string
	Channel Channel
	Signal  *SignalSubscription
	WebPush *WebPushSubscription
}
