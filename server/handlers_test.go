package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func newServer(t *testing.T) (http.Handler, *Board) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "state.json")
	b, err := NewBoard(path)
	if err != nil {
		t.Fatal(err)
	}
	return NewMux(b), b
}

func TestPostCardCreates201(t *testing.T) {
	mux, _ := newServer(t)

	body := `{"title":"Set up DNS","description":"A record","column":"to-do"}`
	req := httptest.NewRequest(http.MethodPost, "/api/cards", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
	var got Card
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("body not JSON: %v\n%s", err, rec.Body.String())
	}
	if got.Title != "Set up DNS" || got.Column != "to-do" || got.ID == "" {
		t.Errorf("unexpected card: %+v", got)
	}
}

func TestPostCardMissingTitleReturns400(t *testing.T) {
	mux, _ := newServer(t)
	body := `{"description":"no title","column":"to-do"}`
	req := httptest.NewRequest(http.MethodPost, "/api/cards", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rec.Code)
	}
}

func TestGetCardsReturnsAll(t *testing.T) {
	mux, b := newServer(t)
	b.AddCard("a", "", "to-do")
	b.AddCard("b", "", "done")

	req := httptest.NewRequest(http.MethodGet, "/api/cards", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	var cards []Card
	if err := json.Unmarshal(rec.Body.Bytes(), &cards); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if len(cards) != 2 {
		t.Errorf("got %d cards, want 2", len(cards))
	}
}

func TestPatchCardUpdates(t *testing.T) {
	mux, b := newServer(t)
	c, _ := b.AddCard("orig", "", "to-do")

	body := `{"column":"done"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/cards/"+c.ID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var got Card
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Column != "done" {
		t.Errorf("column not updated: %+v", got)
	}
}

func TestPatchUnknownIDReturns404(t *testing.T) {
	mux, _ := newServer(t)
	req := httptest.NewRequest(http.MethodPatch, "/api/cards/no-such-id", strings.NewReader(`{"column":"done"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", rec.Code)
	}
}

func TestDeleteCardReturns204(t *testing.T) {
	mux, b := newServer(t)
	c, _ := b.AddCard("doomed", "", "to-do")

	req := httptest.NewRequest(http.MethodDelete, "/api/cards/"+c.ID, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204", rec.Code)
	}
	if len(b.ListCards()) != 0 {
		t.Errorf("card not actually deleted")
	}
}

func TestDeleteUnknownIDReturns404(t *testing.T) {
	mux, _ := newServer(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/cards/no-such-id", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", rec.Code)
	}
}
