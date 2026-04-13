package signal

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"

	qrcode "github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"

	"prism/service/util"
)

type Handlers struct {
	Client *Client
	tmpl   *util.TemplateRenderer
	logger *slog.Logger
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
}

func NewHandlers(client *Client, tmpl *util.TemplateRenderer, logger *slog.Logger) *Handlers {
	return &Handlers{
		Client: client,
		tmpl:   tmpl,
		logger: logger,
	}
}

func (h *Handlers) HandleFragment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	var contentData SignalContentData
	var integData IntegrationData
	integData.Name = "Signal"

	if h.Client == nil {
		integData.StatusClass = "disconnected"
		integData.StatusText = "Not Available"
		integData.StatusTooltip = "signal-cli not found in PATH"
		integData.Open = true
		contentData.Error = "signal-cli not found. Download from: https://github.com/AsamK/signal-cli/releases"
	} else {
		account, err := h.Client.GetLinkedAccount()
		if err != nil {
			integData.StatusClass = "disconnected"
			integData.StatusText = "Error"
			integData.StatusTooltip = err.Error()
			integData.Open = true
			contentData.Error = err.Error()
		} else if account == nil {
			integData.StatusClass = "disconnected"
			integData.StatusText = "Unlinked"
			integData.StatusTooltip = "Click Link button below to link"
			integData.Open = true
		} else {
			integData.StatusClass = "connected"
			integData.StatusText = "Linked"
			integData.StatusTooltip = FormatPhoneNumber(account.Number)
			integData.Open = false
			contentData.Linked = true
			contentData.DeviceName = FormatPhoneNumber(account.Number)
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

func (h *Handlers) HandleLinkDevice(w http.ResponseWriter, r *http.Request) {
	if h.Client == nil {
		util.JSONError(w, "signal-cli not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		DeviceName string `json:"device_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.DeviceName = DefaultDeviceName
	}

	qrCode, err := h.Client.LinkDevice(req.DeviceName)
	if err != nil {
		h.logger.Error("Failed to generate link code", "error", err)
		util.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	qrc, err := qrcode.NewWith(qrCode, qrcode.WithErrorCorrectionLevel(qrcode.ErrorCorrectionMedium))
	if err != nil {
		h.logger.Error("Failed to encode QR code", "error", err)
		util.JSONError(w, "Failed to generate QR code", http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	w2 := standard.NewWithWriter(nopWriteCloser{&buf},
		standard.WithBuiltinImageEncoder(standard.PNG_FORMAT),
		standard.WithQRWidth(10),
	)
	if err := qrc.Save(w2); err != nil {
		h.logger.Error("Failed to render QR code", "error", err)
		util.JSONError(w, "Failed to generate QR code", http.StatusInternalServerError)
		return
	}
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"qr_code": dataURL,
		"status":  "linking",
	})
}

func (h *Handlers) HandleLinkStatus(w http.ResponseWriter, r *http.Request) {
	if h.Client == nil {
		util.JSONError(w, "signal-cli not available", http.StatusServiceUnavailable)
		return
	}

	account, err := h.Client.GetLinkedAccount()
	if err != nil {
		h.logger.Error("Failed to get linked account", "error", err)
		util.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Debug("Link status check", "linked", account != nil, "account", account)

	w.Header().Set("Content-Type", "application/json")
	if account != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"linked":       true,
			"phone_number": account.Number,
		})
	} else {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"linked": false,
		})
	}
}

type nopWriteCloser struct{ *bytes.Buffer }

func (nopWriteCloser) Close() error { return nil }
