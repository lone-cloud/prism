package notification

type Action struct {
	ID       string         `json:"id"`
	Label    string         `json:"label"`
	Endpoint string         `json:"endpoint"`
	Method   string         `json:"method"`
	Data     map[string]any `json:"data,omitempty"`
}

type Notification struct {
	Title   string   `json:"title,omitempty"`
	Message string   `json:"message"`
	Tag     string   `json:"tag,omitempty"`
	Actions []Action `json:"actions,omitempty"`
}

type Channel string

const (
	ChannelSignal   Channel = "signal"
	ChannelWebPush  Channel = "webpush"
	ChannelTelegram Channel = "telegram"
)

func (c Channel) String() string {
	return string(c)
}

func (c Channel) Label() string {
	switch c {
	case ChannelSignal:
		return "Signal"
	case ChannelWebPush:
		return "WebPush"
	case ChannelTelegram:
		return "Telegram"
	default:
		return string(c)
	}
}

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
	Signal  *SignalSubscription
	WebPush *WebPushSubscription
	AppName string
	Channel Channel
}
