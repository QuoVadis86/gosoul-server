package deal

import (
	"context"
	"testing"

	"github.com/qy-info/gosoul/internal/game/engine"
)

func TestRandomWallYonma(t *testing.T) {
	w, err := RandomWallFactory{}.BuildWall(context.Background(), engine.RoundMeta{
		NumPlayers: 4,
	})
	if err != nil {
		t.Fatal(err)
	}

	// 4 hands x 13 + dealer extra + draw wall + 14 dead wall = 136.
	handTiles := 0
	for _, h := range w.Hands {
		if len(h) != 13 {
			t.Fatalf("hand size = %d, want 13", len(h))
		}
		handTiles += len(h)
	}
	total := handTiles + 1 + len(w.Wall) + len(w.DeadWall)
	if total != 136 {
		t.Fatalf("total tiles = %d, want 136", total)
	}

	counts := map[engine.Tile]int{}
	for _, h := range w.Hands {
		for _, t := range h {
			counts[t]++
		}
	}
	counts[w.DealerExtra]++
	for _, t := range w.Wall {
		counts[t]++
	}
	for _, t := range w.DeadWall {
		counts[t]++
	}
	for tile, n := range counts {
		if n != 4 {
			t.Fatalf("tile %s appears %d times, want 4", tile, n)
		}
	}
}

func TestPresetInjectsHand(t *testing.T) {
	store := NewStore()
	store.Upsert(&Preset{
		ID:    "demo",
		Name:  "demo",
		Hands: [][]string{{"1m", "1m", "1m", "2m", "2m", "2m", "3m", "3m", "3m", "4m", "4m", "4m", "5m"}},
	})

	w, err := store.Name("demo").BuildWall(context.Background(), engine.RoundMeta{NumPlayers: 4})
	if err != nil {
		t.Fatal(err)
	}
	if w.Hands[0][0] != "1m" {
		t.Fatalf("seat 0 hand not injected: %v", w.Hands[0])
	}
	if len(w.Hands[0]) != 13 {
		t.Fatalf("seat 0 hand size = %d", len(w.Hands[0]))
	}
}

func TestPresetFallback(t *testing.T) {
	store := NewStore()
	// Missing preset must degrade to random dealing.
	w, err := store.Name("missing").BuildWall(context.Background(), engine.RoundMeta{NumPlayers: 4})
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Hands) != 4 {
		t.Fatalf("hand count = %d", len(w.Hands))
	}
}
