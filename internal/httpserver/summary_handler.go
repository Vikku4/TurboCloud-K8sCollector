package httpserver

import (
	"context"
	"net/http"
	"strconv"

	"github.com/turbocloud-io/k8s-collector/internal/core"
)

func (s *Server) handleGetSummary(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.requestTimeout)
	defer cancel()

	// In production, API Gateway (after JWT validation) injects tenant_id here.
	tenantID := r.Header.Get("X-Tenant-Id")
	if tenantID == "" {
		http.Error(w, "missing X-Tenant-Id header", http.StatusUnauthorized)
		return
	}

	windowStr := r.URL.Query().Get("window_minutes")
	window := 60
	if windowStr != "" {
		if v, err := strconv.Atoi(windowStr); err == nil && v > 0 {
			window = v
		}
	}

	params := core.SummaryParams{
		WindowMinutes: window,
		ClusterID:     r.URL.Query().Get("cluster_id"),
		Namespace:     r.URL.Query().Get("namespace"),
	}

	rows, err := s.svc.GetUsageSummary(ctx, tenantID, params)
	if err != nil {
		s.logger.Errorf("summary error: %v", err)
		http.Error(w, "failed to fetch summary", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"window_minutes": window,
		"items":          rows,
	})
}
