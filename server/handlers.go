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
//	GET    /api/cards            list all cards
//	POST   /api/cards            create
//	PATCH  /api/cards/{id}       sparse update
//	DELETE /api/cards/{id}       remove
//	GET    /api/projects         list all projects
//	POST   /api/projects         create project
//	DELETE /api/projects/{id}    remove project
//	GET    /api/config           server config (agent list etc.)
//	GET    /                     static frontend (served by caller separately)
func NewMux(b *Board, agents []string) http.Handler {
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
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, b.ListProjects())
		case http.MethodPost:
			handleCreateProject(w, r, b)
		default:
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/projects/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/projects/")
		if id == "" {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodDelete:
			handleDeleteProject(w, r, b, id)
		default:
			w.Header().Set("Allow", "DELETE")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"agents": agents})
	})
	return mux
}

type createRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Column      string `json:"column"`
	Color       string `json:"color"`
	ProjectID   string `json:"projectId"`
}

// validColors is the allowlist for the Card.Color field. Empty string
// means "no colour" and is always allowed. Keep this in sync with the
// CSS palette in static/style.css.
var validColors = map[string]struct{}{
	"":       {},
	"red":    {},
	"orange": {},
	"yellow": {},
	"green":  {},
	"blue":   {},
	"purple": {},
	"grey":   {},
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
	if _, ok := validColors[req.Color]; !ok {
		http.Error(w, "invalid color (allowed: red, orange, yellow, green, blue, purple, grey, or empty)", http.StatusBadRequest)
		return
	}
	c, err := b.AddCard(req.Title, req.Description, req.Column, req.Color, req.ProjectID)
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
	if u.Color != nil {
		if _, ok := validColors[*u.Color]; !ok {
			http.Error(w, "invalid color (allowed: red, orange, yellow, green, blue, purple, grey, or empty)", http.StatusBadRequest)
			return
		}
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

type createProjectRequest struct {
	Name string `json:"name"`
}

func handleCreateProject(w http.ResponseWriter, r *http.Request, b *Board) {
	var req createProjectRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	p, err := b.AddProject(strings.TrimSpace(req.Name))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func handleDeleteProject(w http.ResponseWriter, r *http.Request, b *Board, id string) {
	err := b.DeleteProject(id)
	if errors.Is(err, ErrProjectNotFound) {
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
