package util

import (
	"log/slog"
	"net/http"
)

func LogAndError(w http.ResponseWriter, logger *slog.Logger, message string, code int, err error) {
	if err != nil {
		logger.Error(message, "error", err)
	} else {
		logger.Error(message)
	}
	http.Error(w, message, code)
}
