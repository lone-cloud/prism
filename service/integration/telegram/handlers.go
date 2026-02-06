package telegram

import (
	"html/template"
	"log/slog"
	"net/http"

	"prism/service/util"
)

type Handlers struct {
	client *Client
	chatID int64
	tmpl   *util.TemplateRenderer
	logger *slog.Logger
}

type TelegramContentData struct {
	NotConfigured bool
	Error         string
	NeedsChatID   bool
}

type IntegrationData struct {
	Name          string
	StatusClass   string
	StatusText    string
	StatusTooltip string
	Content       template.HTML
	Open          bool
	PollAttrs     string
}

func NewHandlers(client *Client, chatID int64, tmpl *util.TemplateRenderer, logger *slog.Logger) *Handlers {
	return &Handlers{
		client: client,
		chatID: chatID,
		tmpl:   tmpl,
		logger: logger,
	}
}

func (h *Handlers) HandleFragment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	var contentData TelegramContentData
	var integData IntegrationData
	integData.Name = "Telegram"

	if h.client == nil {
		integData.StatusClass = "disconnected"
		integData.StatusText = "Not Configured"
		integData.Open = true
		contentData.NotConfigured = true
	} else {
		bot, err := h.client.GetMe()
		if err != nil {
			integData.StatusClass = "disconnected"
			integData.StatusText = "Error"
			integData.Open = true
			contentData.Error = err.Error()
		} else if h.chatID == 0 {
			integData.StatusClass = "disconnected"
			integData.StatusText = "Needs Chat ID"
			integData.StatusTooltip = "@" + bot.Username
			integData.Open = true
			contentData.NeedsChatID = true
		} else {
			integData.StatusClass = "connected"
			integData.StatusText = "Linked"
			integData.StatusTooltip = "@" + bot.Username
			integData.Open = false
		}
	}

	content, err := h.tmpl.RenderHTML("telegram-content.html", contentData)
	if err != nil {
		util.LogAndError(w, h.logger, "Internal server error", http.StatusInternalServerError, err)
		return
	}
	integData.Content = content

	html, err := h.tmpl.Render("integration.html", integData)
	if err != nil {
		util.LogAndError(w, h.logger, "Internal server error", http.StatusInternalServerError, err)
		return
	}

	w.Write([]byte(html))
}

func (h *Handlers) IsEnabled() bool {
	return true
}

func (h *Handlers) GetClient() *Client {
	return h.client
}
