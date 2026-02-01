package proton

import (
	"fmt"
	"log/slog"

	"github.com/lone-cloud/prism/internal/config"
)

type ActionHandler struct {
	monitor *Monitor
	cfg     *config.Config
	logger  *slog.Logger
}

func NewActionHandler(monitor *Monitor, cfg *config.Config, logger *slog.Logger) *ActionHandler {
	return &ActionHandler{
		monitor: monitor,
		cfg:     cfg,
		logger:  logger,
	}
}

func (h *ActionHandler) HandleAction(actionType, messageID string) error {
	if actionType != h.cfg.ActionMarkRead {
		return fmt.Errorf("unknown action type: %s", actionType)
	}

	if err := h.monitor.MarkAsRead(messageID); err != nil {
		h.logger.Error("Failed to mark message as read", "messageID", messageID, "error", err)
		return fmt.Errorf("failed to mark as read: %w", err)
	}

	h.logger.Info("Marked message as read", "messageID", messageID)
	return nil
}
