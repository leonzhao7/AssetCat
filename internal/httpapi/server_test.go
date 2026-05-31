package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"asset-risk-system/internal/domain"
	"asset-risk-system/internal/httpapi"
	"asset-risk-system/internal/store"
)

func TestAssetLifecycle(t *testing.T) {
	repository, err := store.New("")
	if err != nil {
		t.Fatalf("store.New returned error: %v", err)
	}
	handler := httpapi.New(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))

	createBody := domain.Asset{
		PrimaryDomain: "Example.COM.",
		IPs: []domain.IPRecord{{
			Address: "203.0.113.10",
			Ports: []domain.PortRecord{{
				Port:     443,
				Protocol: "tcp",
				Service:  "https",
				Banner:   "nginx/1.24",
			}},
		}},
		Domains: []domain.DomainRecord{{
			Name: "api.example.com",
			Kind: domain.DomainKindSubdomain,
		}},
		Components: []domain.ComponentRecord{{
			Name:            "nginx",
			Version:         "1.24",
			ProofURL:        "https://example.com/",
			ResponseContent: "HTTP/1.1 200 OK\r\nServer: nginx/1.24\r\n\r\n",
		}},
	}
	created := doJSON[domain.Asset](t, handler, http.MethodPost, "/assets", createBody, http.StatusCreated)
	if created.PrimaryDomain != "example.com" {
		t.Fatalf("PrimaryDomain = %q, want normalized example.com", created.PrimaryDomain)
	}
	if created.ID == "" {
		t.Fatal("created asset id is empty")
	}

	risk := domain.RiskFinding{
		Title:    "admin console exposed",
		Severity: domain.SeverityHigh,
		URL:      "https://api.example.com/admin",
		Request:  "GET /admin HTTP/1.1\r\nHost: api.example.com\r\n\r\n",
		Response: "HTTP/1.1 200 OK\r\n\r\nadmin",
	}
	updated := doJSON[domain.Asset](t, handler, http.MethodPost, "/assets/"+created.ID+"/domains/api.example.com/risks", risk, http.StatusOK)
	if len(updated.Domains) != 2 {
		t.Fatalf("len(updated.Domains) = %d, want primary plus subdomain", len(updated.Domains))
	}

	risks := doJSON[[]domain.RiskFinding](t, handler, http.MethodGet, "/assets/"+created.ID+"/risks?severity=high", nil, http.StatusOK)
	if len(risks) != 1 || risks[0].Title != risk.Title {
		t.Fatalf("risks = %#v, want high severity risk", risks)
	}

	summary := doJSON[domain.AssetSummary](t, handler, http.MethodGet, "/summary", nil, http.StatusOK)
	if summary.Assets != 1 || summary.Risks != 1 || summary.Ports != 1 || summary.Components != 1 {
		t.Fatalf("summary = %#v, want counts for one populated asset", summary)
	}
}

func TestStaticFrontendFallback(t *testing.T) {
	repository, err := store.New("")
	if err != nil {
		t.Fatalf("store.New returned error: %v", err)
	}
	webDir := t.TempDir()
	index := []byte("<!doctype html><html><body>AssetCat UI</body></html>")
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), index, 0o644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}
	handler := httpapi.NewWithStatic(repository, slog.New(slog.NewTextHandler(io.Discard, nil)), webDir)

	for _, path := range []string{"/", "/assets-view/overview"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200", path, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "AssetCat UI") {
			t.Fatalf("GET %s body = %q, want index fallback", path, rec.Body.String())
		}
	}
}

func doJSON[T any](t *testing.T, handler http.Handler, method string, path string, body any, wantStatus int) T {
	t.Helper()
	var reader io.Reader
	if body != nil {
		content, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal returned error: %v", err)
		}
		reader = bytes.NewReader(content)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("%s %s status = %d, want %d, body: %s", method, path, rec.Code, wantStatus, rec.Body.String())
	}

	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("json.Unmarshal response returned error: %v; body: %s", err, rec.Body.String())
	}
	return out
}
