package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"asset-risk-system/internal/domain"
	"asset-risk-system/internal/store"
)

type Server struct {
	store  *store.Store
	logger *slog.Logger
}

func New(store *store.Store, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	server := &Server{store: store, logger: logger}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.health)
	mux.HandleFunc("GET /summary", server.summary)
	mux.HandleFunc("/assets", server.assets)
	mux.HandleFunc("/assets/", server.assetRoutes)
	return requestLog(logger, recoverPanic(jsonMiddleware(mux)))
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) summary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	writeJSON(w, http.StatusOK, s.store.Summary())
}

func (s *Server) assets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listAssets(w, r)
	case http.MethodPost:
		s.createAsset(w, r)
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) assetRoutes(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/assets/"))
	if len(parts) == 0 || parts[0] == "" {
		notFound(w)
		return
	}
	assetID, err := url.PathUnescape(parts[0])
	if err != nil {
		badRequest(w, "invalid asset id")
		return
	}

	if len(parts) == 1 {
		s.assetByID(w, r, assetID)
		return
	}

	switch parts[1] {
	case "domains":
		s.domainRoutes(w, r, assetID, parts[2:])
	case "ips":
		if len(parts) == 2 && r.Method == http.MethodPost {
			s.addIP(w, r, assetID)
			return
		}
		methodNotAllowed(w, http.MethodPost)
	case "components":
		if len(parts) == 2 && r.Method == http.MethodPost {
			s.addComponent(w, r, assetID)
			return
		}
		methodNotAllowed(w, http.MethodPost)
	case "risks":
		if len(parts) == 2 && r.Method == http.MethodGet {
			s.listRisks(w, r, assetID)
			return
		}
		methodNotAllowed(w, http.MethodGet)
	default:
		notFound(w)
	}
}

func (s *Server) assetByID(w http.ResponseWriter, r *http.Request, assetID string) {
	switch r.Method {
	case http.MethodGet:
		asset, ok := s.store.Get(assetID)
		if !ok {
			notFound(w)
			return
		}
		writeJSON(w, http.StatusOK, asset)
	case http.MethodPut:
		var asset domain.Asset
		if !decodeJSON(w, r, &asset) {
			return
		}
		updated, err := s.store.Replace(assetID, asset)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, updated)
	case http.MethodDelete:
		if err := s.store.Delete(assetID); err != nil {
			writeStoreError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPut, http.MethodDelete)
	}
}

func (s *Server) domainRoutes(w http.ResponseWriter, r *http.Request, assetID string, rest []string) {
	if len(rest) == 0 && r.Method == http.MethodPost {
		s.addDomain(w, r, assetID)
		return
	}
	if len(rest) == 2 && rest[1] == "risks" && r.Method == http.MethodPost {
		domainName, err := url.PathUnescape(rest[0])
		if err != nil {
			badRequest(w, "invalid domain name")
			return
		}
		s.addRisk(w, r, assetID, domainName)
		return
	}
	methodNotAllowed(w, http.MethodPost)
}

func (s *Server) listAssets(w http.ResponseWriter, r *http.Request) {
	assets := s.store.List()
	filtered := assets[:0]
	for _, asset := range assets {
		if assetMatches(asset, r.URL.Query()) {
			filtered = append(filtered, asset)
		}
	}
	writeJSON(w, http.StatusOK, filtered)
}

func (s *Server) createAsset(w http.ResponseWriter, r *http.Request) {
	var asset domain.Asset
	if !decodeJSON(w, r, &asset) {
		return
	}
	created, err := s.store.Upsert(asset)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) addDomain(w http.ResponseWriter, r *http.Request, assetID string) {
	var record domain.DomainRecord
	if !decodeJSON(w, r, &record) {
		return
	}
	asset, err := s.store.AddDomain(assetID, record)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, asset)
}

func (s *Server) addIP(w http.ResponseWriter, r *http.Request, assetID string) {
	var record domain.IPRecord
	if !decodeJSON(w, r, &record) {
		return
	}
	asset, err := s.store.AddIP(assetID, record)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, asset)
}

func (s *Server) addComponent(w http.ResponseWriter, r *http.Request, assetID string) {
	var component domain.ComponentRecord
	if !decodeJSON(w, r, &component) {
		return
	}
	asset, err := s.store.AddComponent(assetID, component)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, asset)
}

func (s *Server) addRisk(w http.ResponseWriter, r *http.Request, assetID string, domainName string) {
	var finding domain.RiskFinding
	if !decodeJSON(w, r, &finding) {
		return
	}
	asset, err := s.store.AddRisk(assetID, domainName, finding)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, asset)
}

func (s *Server) listRisks(w http.ResponseWriter, r *http.Request, assetID string) {
	asset, ok := s.store.Get(assetID)
	if !ok {
		notFound(w)
		return
	}
	risks := domain.FlattenRisks(asset)
	severity := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("severity")))
	if severity != "" {
		filtered := risks[:0]
		for _, risk := range risks {
			if string(risk.Severity) == severity {
				filtered = append(filtered, risk)
			}
		}
		risks = filtered
	}
	writeJSON(w, http.StatusOK, risks)
}

func assetMatches(asset domain.Asset, query url.Values) bool {
	q := strings.ToLower(strings.TrimSpace(query.Get("q")))
	ip := strings.TrimSpace(query.Get("ip"))
	component := strings.ToLower(strings.TrimSpace(query.Get("component")))
	severity := strings.ToLower(strings.TrimSpace(query.Get("severity")))

	if q != "" {
		matched := strings.Contains(strings.ToLower(asset.ID), q) ||
			strings.Contains(strings.ToLower(asset.PrimaryDomain), q) ||
			strings.Contains(strings.ToLower(asset.Name), q)
		for _, record := range asset.Domains {
			matched = matched || strings.Contains(strings.ToLower(record.Name), q)
		}
		if !matched {
			return false
		}
	}
	if ip != "" {
		matched := false
		for _, record := range asset.IPs {
			matched = matched || record.Address == ip
		}
		if !matched {
			return false
		}
	}
	if component != "" {
		matched := false
		for _, record := range asset.Components {
			matched = matched ||
				strings.Contains(strings.ToLower(record.Name), component) ||
				strings.Contains(strings.ToLower(record.Version), component)
		}
		if !matched {
			return false
		}
	}
	if severity != "" {
		matched := false
		for _, record := range asset.Domains {
			for _, risk := range record.Risks {
				matched = matched || string(risk.Severity) == severity
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		notFound(w)
	case errors.Is(err, domain.ErrValidation):
		badRequest(w, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
	}
}
