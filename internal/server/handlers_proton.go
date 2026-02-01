package server

import (
	"encoding/json"
	"net/http"
)

type markReadRequest struct {
	UID int `json:"uid"`
}

func (s *Server) handleProtonMarkRead(w http.ResponseWriter, r *http.Request) {
	var req markReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"}) //nolint:errcheck // Error encoding response is not critical
		return
	}

	if req.UID == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "uid (number) is required"}) //nolint:errcheck // Error encoding response is not critical
		return
	}

	// TODO: Implement mark as read using UID
	// For now, this is a placeholder since the IMAP implementation needs the sequence number
	s.logger.Debug("Mark email as read requested", "uid", req.UID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]bool{"success": true}); err != nil {
		s.logger.Error("Failed to encode response", "error", err)
	}
}
