package server

import (
	"net/http"
)

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := struct {
		Version string
	}{
		Version: s.version,
	}
	if err := s.indexTmpl.Execute(w, data); err != nil {
		s.logger.Error("failed to execute index template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
