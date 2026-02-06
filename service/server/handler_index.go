package server

import (
	"net/http"
	"prism/service/util"
)

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.indexTmpl.Execute(w, map[string]string{"Version": s.version}); err != nil {
		util.LogAndError(w, s.logger, "Internal Server Error", http.StatusInternalServerError, err)
	}
}
