package server

import (
	"encoding/json"
	"net/http"
)

type markReadRequest struct {
	UID uint32 `json:"uid"`
}

func (s *Server) handleProtonMarkRead(w http.ResponseWriter, r *http.Request) {
	var req markReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"}) //nolint:errcheck
		return
	}

	if req.UID == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "uid (number) is required"}) //nolint:errcheck
		return
	}

	if err := s.protonMonitor.MarkAsRead(req.UID); err != nil {
		s.logger.Error("failed to mark email as read", "uid", req.UID, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to mark as read"}) //nolint:errcheck
		return
	}

	s.logger.Info("marked email as read", "uid", req.UID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]bool{"success": true}); err != nil {
		s.logger.Error("failed to encode response", "error", err)
	}
}
