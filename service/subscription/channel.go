package subscription

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
