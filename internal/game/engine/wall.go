package engine

import "context"

// RoundMeta describes the parameters of a round about to start.
type RoundMeta struct {
	NumPlayers   int
	InitialScore int // default 25000 (canonical riichi)
	Kyoku        int // 0..3 (round index); dealer = Kyoku % NumPlayers
	Honba        int
	Liqibang     int
	Sanma        bool // three-player (108 tiles, no 2-8m)
	NotenBappu   bool // ryukyoku penalty on/off
}

// Wall is a fully prescribed tile layout for one round. It decouples the game
// engine from where tiles come from: the default source shuffles randomly,
// while the GM console can inject a fixed preset (e.g. a specific hand
// scenario) via a custom WallFactory.
type Wall struct {
	// Hands is 13 tiles per seat, seat order = play order (seat 0 = dealer).
	Hands [][]Tile
	// DealerExtra is the 14th tile dealt to the dealer at round start.
	DealerExtra Tile
	// Wall is the remaining draw stack in draw order.
	Wall []Tile
	// DeadWall holds the final wall tiles that are never drawn.
	// Dora indicators are located at DeadWall positions 4,6,8,10,12;
	// each declared kan reveals the next one.
	DeadWall []Tile
}

// DoraIndicatorAt returns the dora indicator for the n-th revealed indicator
// (0-based), or empty when the dead wall is too short.
func (w *Wall) DoraIndicatorAt(idx int) Tile {
	pos := 4 + idx*2
	if pos >= len(w.DeadWall) {
		return ""
	}
	return w.DeadWall[pos]
}

// UraDoraIndicatorAt returns the ura-dora indicator mirrored below the n-th
// dora indicator.
func (w *Wall) UraDoraIndicatorAt(idx int) Tile {
	pos := 5 + idx*2
	if pos >= len(w.DeadWall) {
		return ""
	}
	return w.DeadWall[pos]
}

// WallFactory constructs the wall for a round. The engine owns no randomness:
// determinism is delegated to the factory, which makes factories the natural
// integration point for GM-configured deal presets and scenario replays.
type WallFactory interface {
	BuildWall(ctx context.Context, meta RoundMeta) (*Wall, error)
}
