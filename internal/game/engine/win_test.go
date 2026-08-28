package engine

import "testing"

func TestCheckWinTanyao(t *testing.T) {
	// 234m 456p 678s 345s + 88p pair = tanyao (all simples, clean sequences)
	hand := []Tile{"2m", "3m", "4m", "4p", "5p", "6p", "6s", "7s", "8s", "3s", "4s", "5s", "8p", "8p"}
	win := CheckWin(0, hand, nil, true, -1, nil)
	if win == nil {
		t.Fatal("tanyao hand should win")
	}
	found := false
	for _, y := range win.Yaku {
		if y == "tanyao" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected tanyao, got %v", win.Yaku)
	}
}

func TestCheckWinNotAWin(t *testing.T) {
	hand := []Tile{"1m", "3m", "5m", "7m", "9m", "1p", "3p", "5p", "7p", "9p", "1s", "3s", "5s", "7s"}
	if w := CheckWin(0, hand, nil, true, -1, nil); w != nil {
		t.Fatalf("random hand should not be a win, got %v", w)
	}
}

func TestCheckWinRon(t *testing.T) {
	// 111m 222p 333s 999s + 55z; ron on 9s (completes the 999s triplet)
	hand := []Tile{"1m", "1m", "1m", "2p", "2p", "2p", "3s", "3s", "3s", "9s", "9s", "9s", "5z", "5z"}
	win := CheckWin(2, hand, nil, false, 3, nil)
	if win == nil {
		t.Fatal("ron hand should win")
	}
	if win.RonFrom != 3 {
		t.Fatalf("ronFrom = %d, want 3", win.RonFrom)
	}
	if win.Seat != 2 {
		t.Fatalf("seat = %d, want 2", win.Seat)
	}
}

func TestCheckWinWithOpenMeld(t *testing.T) {
	open := []Meld{{Type: MeldPon, Tile: "5z", Tiles: []Tile{"5z", "5z", "5z"}, From: 1}}
	// concealed 11 tiles: 123m 456p 789s + 1z pair? need 14 total - 3 open = 11
	hand := []Tile{"1m", "2m", "3m", "4p", "5p", "6p", "7s", "8s", "9s", "1z", "1z"}
	win := CheckWin(1, hand, open, true, -1, nil)
	if win == nil {
		t.Fatal("open-meld hand should win")
	}
}

func TestContainedTiles(t *testing.T) {
	got := ContainedTiles([]Tile{"1m", "2m"}, "3m")
	if len(got) != 3 {
		t.Fatalf("length = %d, want 3", len(got))
	}
}

func TestCheckWinDoraAddsHan(t *testing.T) {
	// honitsu + tsumo hand; indicator 4m points at the 5m in hand.
	hand := []Tile{"1m", "2m", "3m", "4m", "5m", "6m", "7m", "8m", "9m", "5z", "5z", "5z", "1m", "1m"}
	win := CheckWin(0, hand, nil, true, -1, []Tile{"4m", "9z", "9z", "9z", "9z"})
	if win == nil {
		t.Fatal("hand should win")
	}
	if win.Han <= 0 {
		t.Fatalf("expected positive han, got %d", win.Han)
	}
	foundDora := false
	for _, y := range win.Yaku {
		if y == "dora" {
			foundDora = true
		}
	}
	if !foundDora {
		t.Fatalf("expected dora yaku, got %v (han=%d)", win.Yaku, win.Han)
	}
}
