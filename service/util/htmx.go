package util

import (
	"encoding/json"
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
