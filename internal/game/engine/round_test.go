package engine

import (
	"context"
	"errors"
	"testing"
)

type stubDriver struct {
	order  []Tile
	riichi [4]bool
}

func (d *stubDriver) DiscardTile(_ context.Context, v *View) Tile {
	if len(d.order) > 0 {
		t := d.order[0]
		d.order = d.order[1:]
		return t
	}
	if len(v.Hand) > 0 {
		return v.Hand[0]
	}
	panic("no tiles")
}

func (d *stubDriver) DeclareRiichi(_ context.Context, v *View) bool {
	return d.riichi[v.Seat]
}

func newTestRound() (*Round, *stubDriver) {
	w := &Wall{
		Hands: [][]Tile{
			{"1m", "2m", "3m", "4m", "5m", "6m", "7m", "8m", "9m", "1p", "2p", "3p", "4p"},
			{"1s", "2s", "3s", "4s", "5s", "6s", "7s", "8s", "9s", "5z", "6z", "7z", "1z"},
			{"1m", "1m", "1m", "1p", "1p", "1p", "1s", "1s", "1s", "1z", "2z", "3z", "4z"},
			{"9m", "9m", "9m", "9p", "9p", "9p", "9s", "9s", "9s", "5z", "5z", "5z", "6z"},
		},
		DealerExtra: "5z",
		Wall:        []Tile{"1m", "2m", "3p", "4s", "5z", "6z", "7z", "8z"},
		DeadWall:    []Tile{"1m", "2m", "5z", "6z", "1z", "1z", "2z", "2z", "3z", "3z", "4z", "4z", "7z", "7z"},
	}
	d := &stubDriver{}
	r := NewRound(RoundMeta{NumPlayers: 4, InitialScore: 25000, Kyoku: 0}, w, d)
	return r, d
}

func TestRoundDealAndDraw(t *testing.T) {
	r, _ := newTestRound()
	drawn, err := r.Start(context.Background())
	if err != nil || drawn != "5z" {
		t.Fatalf("start: drawn=%v err=%v", drawn, err)
	}
	if len(r.Players[0].Hand) != 14 {
		t.Fatalf("dealer should have 14 tiles after start, has %d", len(r.Players[0].Hand))
	}
	if _, err := r.Draw(context.Background()); err != nil {
		t.Fatalf("draw: %v", err)
	}
	if len(r.Players[0].Hand) != 15 {
		t.Fatalf("dealer hand after draw = %d, want 15", len(r.Players[0].Hand))
	}
	if err := r.Discard(context.Background(), "1m"); err != nil {
		t.Fatalf("discard: %v", err)
	}
	if len(r.Players[0].Hand) != 14 {
		t.Fatalf("dealer hand after discard = %d, want 14", len(r.Players[0].Hand))
	}
	if r.current != 1 {
		t.Fatalf("active seat = %d, want 1", r.current)
	}
}

func TestRoundInvalidDiscard(t *testing.T) {
	r, _ := newTestRound()
	if _, err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Draw(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := r.Discard(context.Background(), "9z"); !errors.Is(err, ErrInvalidDiscard) {
		t.Fatalf("discard 9z = %v, want ErrInvalidDiscard", err)
	}
}

func TestRoundRotation(t *testing.T) {
	r, _ := newTestRound()
	if _, err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		if _, err := r.Draw(context.Background()); err != nil {
			t.Fatalf("draw %d: %v", i, err)
		}
		hand := r.Players[r.current].Hand
		if err := r.Discard(context.Background(), hand[0]); err != nil {
			t.Fatalf("discard %d: %v", i, err)
		}
	}
	if r.current != 0 {
		t.Fatalf("after 8 turns active seat = %d, want 0", r.current)
	}
}

func TestRoundWallExhaustion(t *testing.T) {
	r, _ := newTestRound()
	if _, err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	for r.phase == PhasePlaying {
		_, err := r.Draw(context.Background())
		if err != nil {
			if errors.Is(err, ErrNoTiles) {
				break
			}
			t.Fatalf("unexpected draw error: %v", err)
		}
		hand := r.Players[r.current].Hand
		if err := r.Discard(context.Background(), hand[0]); err != nil {
			t.Fatalf("discard: %v", err)
		}
	}
}

func TestRoundDoraIndicators(t *testing.T) {
	r, _ := newTestRound()
	if len(r.Dora) == 0 {
		t.Fatal("expected dora indicators from dead wall")
	}
	if r.Dora[0] != "1z" {
		t.Fatalf("first dora = %v, want 1z (dead wall offset 4)", r.Dora[0])
	}
}

func TestRoundDefaults(t *testing.T) {
	w := &Wall{Hands: make([][]Tile, 4)}
	d := &stubDriver{}
	r := NewRound(RoundMeta{NumPlayers: 4, Kyoku: 1}, w, d)
	if r.Dealer != 1 {
		t.Fatalf("dealer = %d, want 1 (kyoku 1)", r.Dealer)
	}
	if r.Scores[0] != 25000 {
		t.Fatalf("default score = %d, want 25000", r.Scores[0])
	}
}

func TestDoubleRiichi(t *testing.T) {
	r, _ := newTestRound()
	if _, err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Dealer's first discard declares riichi → daburi.
	r.current = 0
	if err := r.DeclareRiichi(context.Background(), "1m"); err != nil {
		t.Fatal(err)
	}
	if !r.IsDoubleRiichi(0) {
		t.Fatal("first-discard riichi should be double")
	}
	// Rotate: seat 2 has already discarded before declaring → not double.
	r.current = 2
	if _, err := r.Draw(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := r.Discard(context.Background(), "3z"); err != nil {
		t.Fatal(err)
	}
	r.current = 2
	if err := r.DeclareRiichi(context.Background(), "1s"); err != nil {
		t.Fatal(err)
	}
	if r.IsDoubleRiichi(2) {
		t.Fatal("post-discard riichi must not be double")
	}
}
