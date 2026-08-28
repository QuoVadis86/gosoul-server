package engine

import (
	"context"
	"errors"
	"sort"
)

// ErrNotTurn is returned when an operation is applied out of rotation.
var ErrNotTurn = errors.New("engine: not this seat's turn")

// ErrInvalidDiscard is returned when a tile is not in the player's hand.
var ErrInvalidDiscard = errors.New("engine: tile not in hand")

// Phase tracks where in a round we are.
type Phase int

const (
	PhaseDealing Phase = iota
	PhasePlaying
	PhaseEnded
)

// PlayerState is a seat's live hand plus public info during a round.
type PlayerState struct {
	Hand   []Tile
	Melds  []Meld
	Riichi bool
	// Discards tracks every tile this seat has thrown this round.
	Discards []Tile
}

// Round is one kyoku of a match: four players draw and discard until a hand
// resolves or the wall runs out. It consumes a Wall and drives turns through a
// decision callback so both AI bots and human gateways can act.
type Round struct {
	Meta      RoundMeta
	Wall      *Wall
	Dora      []Tile
	DoraCount int

	Players []PlayerState
	Scores  []int
	// RiichiStick counts live riichi stakes.
	RiichiStick int
	// Winner is the seat of the last winner; -1 before resolution.
	Winner   int
	Wind     int
	Kyoku    int
	Honba    int
	Liqibang int
	Dealer   int

	leftWall []Tile
	current  int
	phase    Phase
	// doubleRiichi marks seats that declared daburi this round.
	doubleRiichi []bool
	// decisions is the per-turn driver supplied by the caller.
	decisions DecisionDriver
}

// DecisionDriver lets the loop ask a seat what to do. The engine never guesses
// a choice itself: AI and humans plug in here.
type DecisionDriver interface {
	// DiscardTile asks seat what to throw after drawing.
	DiscardTile(ctx context.Context, v *View) Tile
	// DeclareRiichi asks whether seat wants riichi when eligible.
	DeclareRiichi(ctx context.Context, v *View) bool
}

// NewRound builds a round from a prescribed wall and a decision driver.
func NewRound(meta RoundMeta, w *Wall, d DecisionDriver) *Round {
	init := meta.InitialScore
	if init == 0 {
		init = 25000
	}
	dealer := meta.Kyoku % meta.NumPlayers
	r := &Round{
		Meta:        meta,
		Wall:        w,
		Players:     make([]PlayerState, meta.NumPlayers),
		Scores:      make([]int, meta.NumPlayers),
		RiichiStick: 0,
		Winner:      -1,
		Dealer:      dealer,
		phase:       PhaseDealing,
		decisions:   d,
	}
	for i := range r.Players {
		r.Players[i] = PlayerState{Hand: append([]Tile(nil), w.Hands[i]...), Riichi: false}
		r.Scores[i] = init
	}
	// Dora indicators sit in the dead wall at offsets 4,6,8,...
	for i := 0; i < r.MaxDora(); i++ {
		if t := w.DoraIndicatorAt(i); t != "" {
			r.Dora = append(r.Dora, t)
		}
	}
	r.leftWall = append([]Tile(nil), w.Wall...)
	r.current = dealer
	r.doubleRiichi = make([]bool, meta.NumPlayers)
	return r
}

// MaxDora returns the number of dora indicators available given the mode.
func (r *Round) MaxDora() int { return 5 }

// LeftWall reports how many live tiles remain in the draw stack.
func (r *Round) LeftWall() int { return len(r.leftWall) }

// End marks the round resolved: winner is the winning seat or -1 for ryukyoku.
// It freezes the phase so no further play is accepted.
func (r *Round) End(winner int) {
	r.Winner = winner
	r.phase = PhaseEnded
}

// Current returns the active seat, or -1 after the round ends.
func (r *Round) Current() int {
	if r.phase == PhaseEnded {
		return -1
	}
	return r.current
}

// Start deals the initial 14th tile to the dealer, sets the first draw, and
// moves into the picking phase.
func (r *Round) Start(ctx context.Context) (Tile, error) {
	if r.phase != PhaseDealing {
		return "", ErrNotTurn
	}
	dealerHand := &r.Players[r.Dealer].Hand
	*dealerHand = append(*dealerHand, r.Wall.DealerExtra)
	r.phase = PhasePlaying
	r.current = r.Dealer
	return r.Wall.DealerExtra, nil
}

// ViewFor builds the projection a seat is allowed to see.
func (r *Round) ViewFor(seat int) *View {
	v := &View{
		Seat:           seat,
		Kyoku:          r.Kyoku,
		Honba:          r.Honba,
		RiichiSticks:   r.RiichiStick,
		Dealer:         r.Dealer,
		DoraIndicators: append([]Tile(nil), r.Dora...),
		Scores:         append([]int(nil), r.Scores...),
		Hand:           append([]Tile(nil), r.Players[seat].Hand...),
		Melds:          append([]Meld(nil), r.Players[seat].Melds...),
		DiscardPiles:   make([][]Tile, r.Meta.NumPlayers),
		Riichi:         make([]bool, r.Meta.NumPlayers),
		HasWon:         make([]bool, r.Meta.NumPlayers),
		LeftWall:       len(r.leftWall),
	}
	for i := range r.Players {
		v.DiscardPiles[i] = append([]Tile(nil), r.Players[i].Discards...)
		v.Riichi[i] = r.Players[i].Riichi
	}
	return v
}

// Draw gives the current player their next tile, or returns ErrNoTiles when
// the wall is exhausted (noten end).
var ErrNoTiles = errors.New("engine: wall exhausted")

// Draw pops one tile from the live wall for the current player.
func (r *Round) Draw(ctx context.Context) (Tile, error) {
	if len(r.leftWall) == 0 {
		return "", ErrNoTiles
	}
	t := r.leftWall[0]
	r.leftWall = r.leftWall[1:]
	r.Players[r.current].Hand = append(r.Players[r.current].Hand, t)
	return t, nil
}

// Discard throws a tile from the current seat, recording the public discard
// and switching the active seat to the next player.
func (r *Round) Discard(ctx context.Context, tile Tile) error {
	if r.phase != PhasePlaying {
		return ErrNotTurn
	}
	p := &r.Players[r.current]
	idx := -1
	for i, t := range p.Hand {
		if t == tile {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ErrInvalidDiscard
	}
	p.Hand = append(p.Hand[:idx], p.Hand[idx+1:]...)
	p.Discards = append(p.Discards, tile)
	r.current = (r.current + 1) % r.Meta.NumPlayers
	return nil
}

// DeclareRiichi marks the current seat as riichi if it is closed and discards
// the given tile, banking a stick. A declaration on the seat's very first
// discard before any meld anywhere is a double riichi (daburi).
func (r *Round) DeclareRiichi(ctx context.Context, tile Tile) error {
	p := &r.Players[r.current]
	if len(p.Melds) != 0 {
		return ErrInvalidRiichi
	}
	if err := contains(p.Hand, tile); err != nil {
		return err
	}
	if len(p.Discards) == 0 && !r.anyMeld() {
		r.doubleRiichi[r.current] = true
	}
	p.Riichi = true
	p.Hand = removeTile(p.Hand, tile)
	p.Discards = append(p.Discards, tile)
	r.RiichiStick++
	r.Scores[r.current] -= 1000
	r.current = (r.current + 1) % r.Meta.NumPlayers
	return nil
}

// IsDoubleRiichi reports whether the given seat declared daburi this round.
func (r *Round) IsDoubleRiichi(seat int) bool {
	return r.doubleRiichi[seat]
}

func (r *Round) anyMeld() bool {
	for _, p := range r.Players {
		if len(p.Melds) > 0 {
			return true
		}
	}
	return false
}

// ErrInvalidRiichi is raised when declaring riichi on an open hand.
var ErrInvalidRiichi = errors.New("engine: riichi requires a closed hand")

func contains(hand []Tile, t Tile) error {
	for _, x := range hand {
		if x == t {
			return nil
		}
	}
	return ErrInvalidDiscard
}

func removeTile(hand []Tile, t Tile) []Tile {
	for i, x := range hand {
		if x == t {
			return append(hand[:i], hand[i+1:]...)
		}
	}
	return hand
}

// Effective turns the engine loop: draw, get a decision, and apply it.
// The decision driver chooses between discard and riichi.
func (r *Round) Effective(ctx context.Context) error {
	if r.phase != PhasePlaying {
		return ErrNotTurn
	}
	drawn, err := r.Draw(ctx)
	if err != nil {
		return err
	}
	_ = drawn
	v := r.ViewFor(r.current)
	if r.decisions.DeclareRiichi(ctx, v) {
		tile := r.decisions.DiscardTile(ctx, v)
		return r.DeclareRiichi(ctx, tile)
	}
	tile := r.decisions.DiscardTile(ctx, v)
	return r.Discard(ctx, tile)
}

// SortHand orders a seat's hand canonically.
func (r *Round) SortHand(seat int) {
	sort.Slice(r.Players[seat].Hand, func(i, j int) bool {
		a, b := r.Players[seat].Hand[i], r.Players[seat].Hand[j]
		ai, bi := tileKey(a), tileKey(b)
		return ai < bi
	})
}

func tileKey(t Tile) int {
	switch t[len(t)-1] {
	case 'm':
		return int(t[0]-'0') + 0*9
	case 'p':
		return int(t[0]-'0') + 1*9
	case 's':
		return int(t[0]-'0') + 2*9
	default:
		return int(t[0]-'0') + 3*9
	}
}
