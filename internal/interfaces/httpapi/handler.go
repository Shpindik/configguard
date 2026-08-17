package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"configguard/internal/application/scanner"
	"configguard/internal/infrastructure/parser"
)

type scanResponse struct {
	scanner.Report
	HasIssues bool `json:"has_issues"`
}

// handleScan принимает сырой JSON/YAML конфиг в теле запроса и возвращает
// найденные проблемы. Сам факт проблем — не HTTP-ошибка: 200 отдаётся всегда,
// кроме случаев, когда тело запроса нельзя разобрать.
func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		http.Error(w, "не удалось прочитать тело запроса", http.StatusBadRequest)
		return
	}

	format := formatFromContentType(r.Header.Get("Content-Type"))
	report, err := s.svc.ScanReader(bytes.NewReader(body), "request", format)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(scanResponse{Report: report, HasIssues: report.HasIssues()})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func formatFromContentType(ct string) parser.Format {
	switch {
	case strings.Contains(ct, "yaml"):
		return parser.YAML
	case strings.Contains(ct, "json"):
		return parser.JSON
	default:
		return parser.Unknown
	}
}
