package telegram

import (
	"database/sql"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"

	"prism/service/credentials"
	"prism/service/util"
)

type Handlers struct {
	client *Client
	chatID int64
	tmpl   *util.TemplateRenderer
	logger *slog.Logger
	db     *sql.DB
	apiKey string
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
}

func NewHandlers(client *Client, chatID int64, tmpl *util.TemplateRenderer, logger *slog.Logger) *Handlers {
	return &Handlers{
		client: client,
		chatID: chatID,
		tmpl:   tmpl,
		logger: logger,
		db:     nil,
		apiKey: "",
	}
}

func (h *Handlers) SetDB(db *sql.DB, apiKey string) {
	h.db = db
	h.apiKey = apiKey
}

func (h *Handlers) loadFreshCredentials() (*Client, int64, bool) {
	if h.db == nil || h.apiKey == "" {
		return nil, 0, false
	}

	credStore, err := credentials.NewStore(h.db, h.apiKey)
	if err != nil {
		return nil, 0, false
	}

	creds, err := credStore.GetTelegram()
	if err != nil || creds == nil {
		return nil, 0, false
	}

	client, err := NewClient(creds.BotToken)
	if err != nil {
		h.logger.Error("Failed to create Telegram client", "error", err)
		return nil, 0, false
	}

	var chatID int64
	if creds.ChatID != "" {
		chatID, err = strconv.ParseInt(creds.ChatID, 10, 64)
		if err != nil {
			h.logger.Error("Failed to parse chat ID", "error", err)
			return client, 0, true
		}
	}

	return client, chatID, true
}

func (h *Handlers) HandleFragment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	client, chatID, _ := h.loadFreshCredentials()

	var contentData TelegramContentData
	var integData IntegrationData
	integData.Name = "Telegram"

	if client == nil {
		integData.StatusClass = "disconnected"
		integData.StatusText = "Unlinked"
		integData.StatusTooltip = "Enter bot token to link"
		integData.Open = true
		contentData.NotConfigured = true
	} else {
		bot, err := client.GetMe()
		if err != nil {
			integData.StatusClass = "disconnected"
			integData.StatusText = "Error"
			integData.StatusTooltip = err.Error()
			integData.Open = true
			contentData.Error = err.Error()
		} else if chatID == 0 {
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
	client, _, _ := h.loadFreshCredentials()
	return client
}

func (h *Handlers) GetChatID() int64 {
	_, chatID, _ := h.loadFreshCredentials()
	return chatID
}
