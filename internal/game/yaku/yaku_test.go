package yaku

import "testing"

func TestAnalyzeBasic(t *testing.T) {
	// 123m 456p 789s 123s + 55z pair = 14 tiles
	hand := ParseTiles("123m456p789s123s55z")
	res := Analyze(hand, nil)
	if len(res) == 0 {
		t.Fatal("expected at least one decomposition")
	}
}

func TestAnalyzeOpenMeld(t *testing.T) {
	open := []Mentsu{{Shuntsu: false, Tile: "5z", N: 3, Open: true}}
	hand := ParseTiles("123m456p789s11z")
	res := Analyze(hand, open)
	if len(res) == 0 {
		t.Fatal("open-meld hand should decompose")
	}
	for _, r := range res {
		if len(r.Melds) != 4 {
			t.Fatalf("expected 4 melds, got %d", len(r.Melds))
		}
		openFound := false
		for _, m := range r.Melds {
			if m.Open && m.Tile == "5z" {
				openFound = true
			}
		}
		if !openFound {
			t.Fatal("open pon lost in decomposition")
		}
	}
}

func TestAnalyzeNotWin(t *testing.T) {
	hand := ParseTiles("123m456p789s123m1z")
	if res := Analyze(hand, nil); len(res) != 0 {
		t.Fatalf("unexpected win: %d", len(res))
	}
}

func TestTenpai(t *testing.T) {
	// 123m 456p 789s 11z 44z = 13 tiles; shanpon waits on 1z/4z.
	hand := ParseTiles("123m456p789s11z44z")
	if len(hand) != 13 {
		t.Fatalf("fixture has %d tiles, want 13", len(hand))
	}
	tp := Tenpai(hand, nil)
	if tp == nil {
		t.Fatal("expected tenpai waits")
	}
	if _, ok := tp["1z"]; !ok {
		t.Fatalf("expected 1z wait, got %v", keysOf(tp))
	}
	if _, ok := tp["4z"]; !ok {
		t.Fatalf("expected 4z wait, got %v", keysOf(tp))
	}
}

func keysOf(m map[T]struct{}) []T {
	var out []T
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestCalcChiitoitsu(t *testing.T) {
	hand := ParseTiles("11m22p33s44z55z66m77p")
	res := Analyze(hand, nil)
	if len(res) == 0 {
		t.Fatal("chiitoitsu should decompose")
	}
	fr := Calc(res[0], hand, &ctx{Zimo: true, Menzen: true})
	found := false
	for _, f := range fr.Fans {
		if f.Name == "chiitoitsu" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected chiitoitsu yaku, got %+v", fr.Fans)
	}
}

func TestCalcKokushi(t *testing.T) {
	hand := ParseTiles("19m19p19s1234567z1z")
	res := Analyze(hand, nil)
	if len(res) == 0 {
		t.Fatal("kokushi should decompose")
	}
	fr := Calc(res[0], hand, &ctx{Zimo: true, Menzen: true})
	if !fr.Yakuman || fr.Fans[0].Name != "kokushi" {
		t.Fatalf("expected kokushi, got %+v", fr.Fans)
	}
}

func TestCalcTanyao(t *testing.T) {
	// 234m 456p 345s 567s + 22m pair (all simples)
	full := ParseTiles("234m456p345s567s22m")
	res := Analyze(full, nil)
	if len(res) == 0 {
		t.Fatal("tanyao hand should decompose")
	}
	fr := Calc(res[0], full, &ctx{TanyaoOK: true, Menzen: true})
	found := false
	for _, f := range fr.Fans {
		if f.Name == "tanyao" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected tanyao, got %+v", fr.Fans)
	}
}

func TestCalcYakuhai(t *testing.T) {
	// 123m 555z(triplet) 789p 123s + 11m pair
	full := ParseTiles("123m555z789p123s11m")
	res := Analyze(full, nil)
	if len(res) == 0 {
		t.Fatal("yakuhai hand should decompose")
	}
	fr := Calc(res[0], full, &ctx{RoundWind: 0, SeatWind: 0, Menzen: true})
	found := false
	for _, f := range fr.Fans {
		if f.Name == "haku" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected haku yaku, got %+v", fr.Fans)
	}
}

func TestSort(t *testing.T) {
	hand := ParseTiles("9s7s2m5p1z0m")
	SortTiles(hand)
	want := "2m 0m 5p 7s 9s 1z"
	if TileString(hand) != want {
		t.Fatalf("got %q want %q", TileString(hand), want)
	}
}

func TestRank(t *testing.T) {
	if ParseTiles("0m")[0].Rank() != 5 {
		t.Fatal("red five should rank 5")
	}
	if ParseTiles("9s")[0].Next() != "1s" {
		t.Fatal("9s dora next should be 1s")
	}
	if ParseTiles("7z")[0].Next() != "1z" {
		t.Fatal("7z dora next should be 1z")
	}
}

func TestNextWraps(t *testing.T) {
	cases := map[string]string{
		"1z": "2z", "4z": "5z", "7z": "1z",
		"1m": "2m", "9m": "1m", "5p": "6p",
	}
	for in, want := range cases {
		if got := ParseTiles(in)[0].Next(); got != T(want) {
			t.Fatalf("%s next = %s, want %s", in, got, want)
		}
	}
}
