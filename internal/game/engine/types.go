package engine

// Tile is a canonical mahjong tile: "1m".."9m", "1p".."9p", "1s".."9s",
// "1z".."7z", plus red fives "0m"/"0p"/"0s".
type Tile string

type MeldType int

const (
	MeldChi MeldType = iota
	MeldPon
	MeldDaiminkan
	MeldAnkan
	MeldKakan
)

// Meld is a claimed group attached to a seat.
type Meld struct {
	Type  MeldType
	Tile  Tile
	Tiles []Tile
	From  int
}

// CallOp is one candidate meld offered after an opponent's discard.
type CallOp struct {
	Type  MeldType
	Tile  Tile
	Combo []Tile
}

// SelfOp enumerates actions a player may take on their own turn.
type SelfOp int

const (
	OpDiscard SelfOp = iota
	OpRiichi
	OpAnkan
	OpKakan
	OpTsumo
)

// View is the immutable per-seat game state projection handed to decision
// makers (AI bots, external models, observers). It contains only information
// that seat is allowed to see.
type View struct {
	Seat           int
	Kyoku          int
	Honba          int
	RiichiSticks   int
	Dealer         int
	DoraIndicators []Tile
	Scores         []int

	Hand  []Tile
	Melds []Meld

	DiscardPiles [][]Tile
	Riichi       []bool
	HasWon       []bool
	LeftWall     int
}

// TileSet is a helper for tile counting.
type TileSet map[Tile]int

// Copy returns a defensive copy of the view.
func (v *View) Copy() *View {
	n := *v
	n.Hand = append([]Tile(nil), v.Hand...)
	n.DoraIndicators = append([]Tile(nil), v.DoraIndicators...)
	n.Scores = append([]int(nil), v.Scores...)
	n.DiscardPiles = make([][]Tile, len(v.DiscardPiles))
	for i, p := range v.DiscardPiles {
		n.DiscardPiles[i] = append([]Tile(nil), p...)
	}
	n.Melds = append([]Meld(nil), v.Melds...)
	return &n
}
