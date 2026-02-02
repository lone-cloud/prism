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
	ChannelSignal  Channel = "signal"
	ChannelWebhook Channel = "webhook"
)

type Mapping struct {
	Endpoint   string
	AppName    string
	Channel    Channel
	GroupID    *string
	UpEndpoint *string
}
