package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Card is one item on the kanban board.
type Card struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Column      string `json:"column"`
	Position    int    `json:"position"`
	// Color is an optional palette tag rendered by the frontend as a
	// left-border + tinted background. Empty string = no colour.
	// Allowed values are validated by the API layer (see handlers.go).
	Color string `json:"color,omitempty"`
}

// Board owns the in-memory state and the JSON state file.
// Mutations write through to disk atomically.
type Board struct {
	path string

	mu    sync.Mutex
	cards []Card
}

// NewBoard loads (or creates) the board state at path.
// A missing file is treated as an empty board.
func NewBoard(path string) (*Board, error) {
	b := &Board{path: path}
	if err := b.load(); err != nil {
		return nil, err
	}
	return b, nil
}

// load reads the state file into b.cards. Missing file = empty.
func (b *Board) load() error {
	data, err := os.ReadFile(b.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read state: %w", err)
	}
	if len(data) == 0 {
		return nil
	}
	var cards []Card
	if err := json.Unmarshal(data, &cards); err != nil {
		return fmt.Errorf("parse state: %w", err)
	}
	b.cards = cards
	return nil
}

// save writes the current in-memory state to disk atomically.
// Caller must hold b.mu.
func (b *Board) save() error {
	if err := os.MkdirAll(filepath.Dir(b.path), 0o755); err != nil {
		return fmt.Errorf("mkdir state dir: %w", err)
	}
	data, err := json.MarshalIndent(b.cards, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	tmp := b.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := os.Rename(tmp, b.path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// ListCards returns a copy of all cards (caller-safe to mutate).
func (b *Board) ListCards() []Card {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Card, len(b.cards))
	copy(out, b.cards)
	return out
}

// AddCard creates a card on the board and persists.
func (b *Board) AddCard(title, description, column, color string) (Card, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	c := Card{
		ID:          newID(),
		Title:       title,
		Description: description,
		Column:      column,
		Color:       color,
	}
	b.cards = append(b.cards, c)
	if err := b.save(); err != nil {
		// Roll back the in-memory append so disk and memory stay consistent.
		b.cards = b.cards[:len(b.cards)-1]
		return Card{}, err
	}
	return c, nil
}

// CardUpdate is a sparse update: any non-nil field is applied to the card,
// nil fields are left alone.
type CardUpdate struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Column      *string `json:"column,omitempty"`
	Position    *int    `json:"position,omitempty"`
	Color       *string `json:"color,omitempty"`
}

// ErrCardNotFound is returned when a card ID doesn't exist on the board.
var ErrCardNotFound = errors.New("card not found")

// UpdateCard applies a sparse update and persists. Unknown IDs return ErrCardNotFound.
func (b *Board) UpdateCard(id string, u CardUpdate) (Card, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i := range b.cards {
		if b.cards[i].ID != id {
			continue
		}
		if u.Title != nil {
			b.cards[i].Title = *u.Title
		}
		if u.Description != nil {
			b.cards[i].Description = *u.Description
		}
		if u.Column != nil {
			b.cards[i].Column = *u.Column
		}
		if u.Position != nil {
			b.cards[i].Position = *u.Position
		}
		if u.Color != nil {
			b.cards[i].Color = *u.Color
		}
		updated := b.cards[i]
		if err := b.save(); err != nil {
			return Card{}, err
		}
		return updated, nil
	}
	return Card{}, ErrCardNotFound
}

// DeleteCard removes a card by ID and persists. Unknown IDs return ErrCardNotFound.
func (b *Board) DeleteCard(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i := range b.cards {
		if b.cards[i].ID != id {
			continue
		}
		b.cards = append(b.cards[:i], b.cards[i+1:]...)
		return b.save()
	}
	return ErrCardNotFound
}

// newID returns an unguessable 16-hex-char ID.
func newID() string {
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	return hex.EncodeToString(buf[:])
}
