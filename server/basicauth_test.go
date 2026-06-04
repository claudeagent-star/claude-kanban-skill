package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "downstream-ok")
	})
}

func TestBasicAuth_NoCredsConfigured_PassesThrough(t *testing.T) {
	h := requireBasicAuth("", "", okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/cards", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	if rr.Body.String() != "downstream-ok" {
		t.Fatalf("downstream not called, body=%q", rr.Body.String())
	}
}

func TestBasicAuth_CredsConfigured_NoHeader_Returns401(t *testing.T) {
	h := requireBasicAuth("pitchforks", "bumble", okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/cards", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
	wa := rr.Header().Get("WWW-Authenticate")
	if !strings.HasPrefix(wa, "Basic ") {
		t.Fatalf("want WWW-Authenticate: Basic ..., got %q", wa)
	}
	if strings.Contains(rr.Body.String(), "downstream-ok") {
		t.Fatalf("downstream should not have run")
	}
}

func TestBasicAuth_WrongPassword_Returns401(t *testing.T) {
	h := requireBasicAuth("pitchforks", "bumble", okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/cards", nil)
	req.SetBasicAuth("pitchforks", "wrong")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestBasicAuth_WrongUser_Returns401(t *testing.T) {
	h := requireBasicAuth("pitchforks", "bumble", okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/cards", nil)
	req.SetBasicAuth("evil", "bumble")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestBasicAuth_CorrectCreds_PassesThrough(t *testing.T) {
	h := requireBasicAuth("pitchforks", "bumble", okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/cards", nil)
	req.SetBasicAuth("pitchforks", "bumble")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	if rr.Body.String() != "downstream-ok" {
		t.Fatalf("downstream not called, body=%q", rr.Body.String())
	}
}
