package server

import (
	"bytes"
	"net/http"

	"prism/service/util"
)

func (s *Server) handleFragmentIntegrations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	var buf bytes.Buffer
	if err := s.fragmentTmpl.ExecuteTemplate(&buf, "integrations.html", s.cfg); err != nil {
		util.LogAndError(w, s.logger, "Internal server error", http.StatusInternalServerError, err)
		return
	}

	w.Write(buf.Bytes())
}
