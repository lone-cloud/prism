package delivery

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
