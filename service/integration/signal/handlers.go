package signal

import (
	"html/template"
	"log/slog"
	"net/http"

	"prism/service/util"
)

type Handlers struct {
	client     *Client
	linkDevice *LinkDevice
	tmpl       *util.TemplateRenderer
	logger     *slog.Logger
}

type SignalContentData struct {
	Linked     bool
	DeviceName string
	Error      string
	QRCode     string
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

func NewHandlers(client *Client, linkDevice *LinkDevice, tmpl *util.TemplateRenderer, logger *slog.Logger) *Handlers {
	return &Handlers{
		client:     client,
		linkDevice: linkDevice,
		tmpl:       tmpl,
		logger:     logger,
	}
}

func (h *Handlers) HandleFragment(w http.ResponseWriter, r *http.Request) {
	if h.client == nil {
		return
	}

	w.Header().Set("Content-Type", "text/html")

	account, _ := h.client.GetLinkedAccount()

	var contentData SignalContentData
	var integData IntegrationData
	integData.Name = "Signal"

	if account != nil {
		integData.StatusClass = "connected"
		integData.StatusText = "Linked"
		integData.StatusTooltip = FormatPhoneNumber(account.Number)
		integData.Open = false
		integData.PollAttrs = ""

		contentData.Linked = true
		contentData.DeviceName = h.linkDevice.deviceName
	} else {
		integData.StatusClass = "unlinked"
		integData.StatusText = "Unlinked"
		integData.Open = true
		integData.PollAttrs = ""

		contentData.Linked = false
		qrCode, err := h.linkDevice.GenerateQR()
		if err != nil {
			contentData.Error = err.Error()
		} else {
			contentData.QRCode = qrCode
		}
	}

	content, err := h.tmpl.RenderHTML("signal-content.html", contentData)
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
	return h.client != nil
}

func (h *Handlers) GetClient() *Client {
	return h.client
}
