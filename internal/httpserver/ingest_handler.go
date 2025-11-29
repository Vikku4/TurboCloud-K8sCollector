package httpserver

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/turbocloud-io/k8s-collector/internal/core"
)

func (s *Server) handleIngestMetrics(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout)
	defer cancel()

	clusterToken := r.Header.Get("X-Cluster-Token")
	if clusterToken == "" {
		http.Error(w, "missing X-Cluster-Token header", http.StatusUnauthorized)
		return
	}

	var req core.IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if err := s.svc.IngestMetricsBatch(ctx, clusterToken, req); err != nil {
		if err == core.ErrInvalidToken {
			http.Error(w, "invalid cluster token", http.StatusUnauthorized)
			return
		}
		s.logger.Errorf("ingest error: %v", err)
		http.Error(w, "failed to ingest metrics", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":             "ok",
		"workloads_ingested": len(req.Workloads),
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
