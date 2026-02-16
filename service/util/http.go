package util

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func SetToast(w http.ResponseWriter, message, toastType string) {
	trigger := map[string]interface{}{
		"showToast": map[string]string{
			"message": message,
			"type":    toastType,
		},
	}
	if data, err := json.Marshal(trigger); err == nil {
		w.Header().Set("HX-Trigger", string(data))
	}
}

func LogAndError(w http.ResponseWriter, logger *slog.Logger, message string, code int, err error, attrs ...any) {
	logAttrs := append([]any{"error", err}, attrs...)
	logger.Error(message, logAttrs...)
	http.Error(w, message, code)
}

func JSONError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
