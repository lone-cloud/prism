package telegram

import (
	"fmt"
	"net/http"
)

type Handlers struct {
	client *Client
	chatID int64
}

func NewHandlers(client *Client, chatID int64) *Handlers {
	return &Handlers{
		client: client,
		chatID: chatID,
	}
}

func (h *Handlers) HandleFragment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	var content string
	var statusBadge string
	var openAttr string

	if h.client == nil {
		statusBadge = `<span class="integration-status disconnected">Not Configured</span>`
		content = `
			<p><strong>Setup Instructions:</strong></p>
			<ol class="link-instructions">
				<li>Message <a href="https://t.me/BotFather" target="_blank">@BotFather</a> on Telegram</li>
				<li>Send <code>/newbot</code> and follow the prompts</li>
				<li>Copy the bot token and add to <code>.env</code>: <code>TELEGRAM_BOT_TOKEN=your-token</code></li>
				<li>Message <a href="https://t.me/userinfobot" target="_blank">@userinfobot</a> to get your Chat ID</li>
				<li>Add to <code>.env</code>: <code>TELEGRAM_CHAT_ID=your-chat-id</code></li>
				<li>Restart Prism</li>
			</ol>
			<p class="text-muted">See <a href="https://github.com/lone-cloud/prism#telegram" target="_blank">full setup guide</a></p>
		`
		openAttr = " open"
	} else {
		bot, err := h.client.GetMe()
		if err != nil {
			statusBadge = `<span class="integration-status disconnected">Error</span>`
			content = fmt.Sprintf(`<p>Error: %s</p>`, err)
			openAttr = " open"
		} else if h.chatID == 0 {
			statusBadge = fmt.Sprintf(`<span class="integration-status disconnected">Needs Chat ID<span class="tooltip">@%s</span></span>`, bot.Username)
			content = `
				<p><strong>Complete Setup:</strong></p>
				<ol class="link-instructions">
					<li>Message <a href="https://t.me/userinfobot" target="_blank">@userinfobot</a> on Telegram to get your Chat ID</li>
					<li>Add to <code>.env</code>: <code>TELEGRAM_CHAT_ID=your-chat-id</code></li>
					<li>Restart Prism</li>
				</ol>
			`
			openAttr = " open"
		} else {
			statusBadge = fmt.Sprintf(`<span class="integration-status connected">Linked<span class="tooltip">@%s</span></span>`, bot.Username)
			content = `
				<p><strong>Unlink Instructions:</strong></p>
				<ol class="link-instructions">
					<li>Remove <code>TELEGRAM_BOT_TOKEN</code> and <code>TELEGRAM_CHAT_ID</code> from <code>.env</code></li>
					<li>Restart: <code>docker compose restart prism</code></li>
				</ol>
			`
			openAttr = ""
		}
	}

	html := fmt.Sprintf(`<details class="integration-card"%s>
		<summary class="integration-header">
			<span class="integration-name">Telegram</span>
			%s
		</summary>
		<div class="integration-content">%s</div>
	</details>`, openAttr, statusBadge, content)

	_, _ = fmt.Fprint(w, html)
}

func (h *Handlers) IsEnabled() bool {
	return true
}

func (h *Handlers) GetClient() *Client {
	return h.client
}
