package engine

import (
	"github.com/qy-info/gosoul/internal/game/yaku"
)

// Win is the resolved outcome of a winning hand.
type Win struct {
	Seat    int
	Tsumo   bool
	Han     int
	Fu      int
	Yaku    []string
	Yakuman bool
	// RonFrom is the seat that discarded the winning tile (-1 for tsumo).
	RonFrom int
}

// CheckWin evaluates whether the given player's hand wins with the provided
// winning tile in hand. The hand must be the full 14 tiles including the
// winning tile for ron, or 13 concealed tiles plus the drawn tile for tsumo.
// melds lists that seat's called groups (nil for a closed hand).
func CheckWin(seat int, hand []Tile, melds []Meld, tsumo bool, ronFrom int) *Win {
	ytiles := make([]yaku.T, 0, len(hand))
	for _, t := range hand {
		ytiles = append(ytiles, yaku.T(t))
	}
	open := make([]yaku.Mentsu, 0, len(melds))
	for _, m := range melds {
		open = append(open, yaku.Mentsu{
			Shuntsu: m.Type == MeldChi,
			Tile:    yaku.T(m.Tile),
			N:       3,
			Open:    true,
		})
	}
	results := yaku.Analyze(ytiles, open)
	if len(results) == 0 {
		return nil
	}
	// Use the first decomposition for scoring (a hand may decompose multiple ways).
	res := results[0]
	ctx := &yaku.Ctx{
		Zimo:      tsumo,
		Menzen:    len(open) == 0,
		SeatWind:  seat % 4,
		RoundWind: 0,
	}
	// Best-effort fu from the winning shape.
	fr := yaku.Calc(res, ytiles, ctx)
	win := &Win{
		Seat:    seat,
		Tsumo:   tsumo,
		Han:     fr.Total,
		Yakuman: fr.Yakuman,
		RonFrom: ronFrom,
	}
	for _, f := range fr.Fans {
		win.Yaku = append(win.Yaku, f.Name)
	}
	win.Fu = yaku.Fu(res, ctx, !tsumo && len(open) == 0)
	return win
}

// ContainedTiles reconciles a 13-tile concealed hand plus the winning tile
// into the 14-tile shape Analyze expects.
func ContainedTiles(concealed []Tile, winTile Tile) []Tile {
	out := make([]Tile, 0, len(concealed)+1)
	out = append(out, concealed...)
	out = append(out, winTile)
	return out
}
