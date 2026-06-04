package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// NewMux wires the kanban routes to a Board.
// Routes:
//
//	GET    /api/cards         list all cards
//	POST   /api/cards         create
//	PATCH  /api/cards/{id}    sparse update
//	DELETE /api/cards/{id}    remove
//	GET    /                  static frontend (served by caller separately)
func NewMux(b *Board) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/cards", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, b.ListCards())
		case http.MethodPost:
			handleCreate(w, r, b)
		default:
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/cards/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/cards/")
		if id == "" {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodPatch:
			handleUpdate(w, r, b, id)
		case http.MethodDelete:
			handleDelete(w, r, b, id)
		default:
			w.Header().Set("Allow", "PATCH, DELETE")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	return mux
}

type createRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Column      string `json:"column"`
}

func handleCreate(w http.ResponseWriter, r *http.Request, b *Board) {
	var req createRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}
	if req.Column == "" {
		req.Column = "to-do"
	}
	c, err := b.AddCard(req.Title, req.Description, req.Column)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func handleUpdate(w http.ResponseWriter, r *http.Request, b *Board, id string) {
	var u CardUpdate
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&u); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	c, err := b.UpdateCard(id, u)
	if errors.Is(err, ErrCardNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func handleDelete(w http.ResponseWriter, r *http.Request, b *Board, id string) {
	err := b.DeleteCard(id)
	if errors.Is(err, ErrCardNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
