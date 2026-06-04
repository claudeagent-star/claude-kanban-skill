package main

import (
	"path/filepath"
	"testing"
)

func freshBoard(t *testing.T) *Board {
	t.Helper()
	path := filepath.Join(t.TempDir(), "state.json")
	b, err := NewBoard(path)
	if err != nil {
		t.Fatalf("NewBoard: %v", err)
	}
	return b
}

func TestNewBoardOnMissingFileStartsEmpty(t *testing.T) {
	b := freshBoard(t)
	if got := len(b.ListCards()); got != 0 {
		t.Fatalf("expected empty board, got %d cards", got)
	}
}

func TestAddCardAppearsInListCards(t *testing.T) {
	b := freshBoard(t)
	c, _ := b.AddCard("first", "", "to-do", "")
	cards := b.ListCards()
	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}
	if cards[0].ID != c.ID {
		t.Errorf("ListCards returned different card: got %+v, want %+v", cards[0], c)
	}
}

func TestAddCardPersistsAcrossReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	// Session 1: create + add.
	b1, _ := NewBoard(path)
	c, _ := b1.AddCard("survives restart", "", "in-progress", "")

	// Session 2: fresh Board on the same path, should see the card.
	b2, err := NewBoard(path)
	if err != nil {
		t.Fatalf("NewBoard reload: %v", err)
	}
	cards := b2.ListCards()
	if len(cards) != 1 {
		t.Fatalf("expected 1 card after reload, got %d", len(cards))
	}
	if cards[0].ID != c.ID || cards[0].Title != "survives restart" || cards[0].Column != "in-progress" {
		t.Errorf("reloaded card mismatch: got %+v", cards[0])
	}
}

func TestUpdateCardChangesFields(t *testing.T) {
	b := freshBoard(t)
	orig, _ := b.AddCard("title", "desc", "to-do", "")

	newTitle := "renamed"
	newCol := "in-progress"
	updated, err := b.UpdateCard(orig.ID, CardUpdate{
		Title:  &newTitle,
		Column: &newCol,
	})
	if err != nil {
		t.Fatalf("UpdateCard: %v", err)
	}
	if updated.Title != "renamed" {
		t.Errorf("Title: got %q, want %q", updated.Title, "renamed")
	}
	if updated.Column != "in-progress" {
		t.Errorf("Column: got %q, want %q", updated.Column, "in-progress")
	}
	if updated.Description != "desc" {
		t.Errorf("Description should be unchanged, got %q", updated.Description)
	}

	// Reload to verify persistence.
	b2, _ := NewBoard(b.path)
	got := b2.ListCards()[0]
	if got.Title != "renamed" || got.Column != "in-progress" {
		t.Errorf("update did not persist: %+v", got)
	}
}

func TestUpdateCardUnknownIDReturnsError(t *testing.T) {
	b := freshBoard(t)
	_, err := b.UpdateCard("does-not-exist", CardUpdate{})
	if err == nil {
		t.Fatalf("expected error for unknown id, got nil")
	}
}

func TestDeleteCardRemoves(t *testing.T) {
	b := freshBoard(t)
	a, _ := b.AddCard("a", "", "to-do", "")
	bcard, _ := b.AddCard("b", "", "done", "")

	if err := b.DeleteCard(a.ID); err != nil {
		t.Fatalf("DeleteCard: %v", err)
	}
	cards := b.ListCards()
	if len(cards) != 1 {
		t.Fatalf("expected 1 card left, got %d", len(cards))
	}
	if cards[0].ID != bcard.ID {
		t.Errorf("wrong card survived")
	}

	// Persistence.
	b2, _ := NewBoard(b.path)
	if len(b2.ListCards()) != 1 {
		t.Errorf("delete did not persist")
	}
}

func TestDeleteCardUnknownIDReturnsError(t *testing.T) {
	b := freshBoard(t)
	if err := b.DeleteCard("nope"); err == nil {
		t.Fatalf("expected error for unknown id")
	}
}

func TestAddCardReturnsCardWithGivenFields(t *testing.T) {
	b := freshBoard(t)
	c, err := b.AddCard("Set up DNS", "A record for kanban.pitchforks.net", "to-do", "")
	if err != nil {
		t.Fatalf("AddCard: %v", err)
	}
	if c.ID == "" {
		t.Errorf("Card ID should be non-empty")
	}
	if c.Title != "Set up DNS" {
		t.Errorf("Title: got %q, want %q", c.Title, "Set up DNS")
	}
	if c.Description != "A record for kanban.pitchforks.net" {
		t.Errorf("Description: got %q", c.Description)
	}
	if c.Column != "to-do" {
		t.Errorf("Column: got %q, want %q", c.Column, "to-do")
	}
}

func TestColorRoundTripsThroughAddAndUpdate(t *testing.T) {
	b := freshBoard(t)
	c, err := b.AddCard("with colour", "", "to-do", "blue")
	if err != nil {
		t.Fatalf("AddCard: %v", err)
	}
	if c.Color != "blue" {
		t.Errorf("Color on create: got %q, want blue", c.Color)
	}

	// Sparse update changes the colour but leaves other fields alone.
	newColor := "red"
	updated, err := b.UpdateCard(c.ID, CardUpdate{Color: &newColor})
	if err != nil {
		t.Fatalf("UpdateCard: %v", err)
	}
	if updated.Color != "red" {
		t.Errorf("Color after update: got %q, want red", updated.Color)
	}
	if updated.Title != "with colour" {
		t.Errorf("Title should be untouched by colour-only update: got %q", updated.Title)
	}

	// Clearing the colour via empty-string update.
	empty := ""
	cleared, err := b.UpdateCard(c.ID, CardUpdate{Color: &empty})
	if err != nil {
		t.Fatalf("UpdateCard clear: %v", err)
	}
	if cleared.Color != "" {
		t.Errorf("Color after clear: got %q, want empty", cleared.Color)
	}

	// Reload from disk to confirm the colour persists.
	b2, err := NewBoard(b.path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	roundtripped, _ := b2.AddCard("round-trip", "", "to-do", "purple")
	all := b2.ListCards()
	if len(all) != 2 {
		t.Fatalf("expected 2 cards after reload, got %d", len(all))
	}
	for _, x := range all {
		if x.ID == roundtripped.ID && x.Color != "purple" {
			t.Errorf("purple lost across reload: got %q", x.Color)
		}
	}
}
