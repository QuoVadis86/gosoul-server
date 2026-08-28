// Package deal controls how a round's wall is built. By default walls are
// shuffled randomly; the GM console can register named presets that fix
// specific seat hands, prefix the draw stack, or prescribe dora indicators.
// The game engine consumes deal sources through the engine.WallFactory
// interface, so the core never depends on this package.
package deal

import (
	"context"
	"math/rand"

	"github.com/qy-info/gosoul/internal/game/engine"
)

// RandomWallFactory shuffles a fresh wall every round. It is the default source.
type RandomWallFactory struct{}

func (RandomWallFactory) BuildWall(_ context.Context, meta engine.RoundMeta) (*engine.Wall, error) {
	return buildShuffledWall(meta), nil
}

// buildShuffledWall constructs a legal wall for the given meta. Tile ordering
// follows the standard ripple: 34 tile kinds x4 (yonma) or 27 kinds x4 (sanma,
// dropping 2m..8m).
func buildShuffledWall(meta engine.RoundMeta) *engine.Wall {
	deck := buildDeck(meta)
	rand.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })

	const deadWallSize = 14
	w := &engine.Wall{}
	drawCount := len(deck) - deadWallSize
	// Deal round-robin: each seat receives handSize tiles in a single pass.
	w.Hands = make([][]engine.Tile, meta.NumPlayers)
	handSize := 13
	k := 0
	for i := range w.Hands {
		hand := make([]engine.Tile, 0, handSize)
		for j := 0; j < handSize; j++ {
			hand = append(hand, deck[k])
			k++
		}
		w.Hands[i] = hand
	}
	w.DealerExtra = deck[k]
	k++

	// Remaining wall includes the draw stack plus the trailing dead wall.
	w.Wall = append(w.Wall, deck[k:drawCount]...)
	w.DeadWall = append(w.DeadWall, deck[drawCount:]...)
	return w
}

func tileKind(n byte, suit byte) engine.Tile {
	return engine.Tile(string([]byte{'0' + n, suit}))
}

// buildDeck returns one legal set of tiles for the mode.
func buildDeck(meta engine.RoundMeta) []engine.Tile {
	var kinds []engine.Tile
	for _, suit := range []byte("mps") {
		hi := byte(9)
		if meta.Sanma {
			hi = 9 // sanma still keeps 1-p; only manzu 2-8 are removed
		}
		for n := byte(1); n <= hi; n++ {
			kinds = append(kinds, tileKind(n, suit))
		}
	}
	if meta.Sanma {
		// 3-player removes manzu 2..8.
		filtered := kinds[:0]
		for _, t := range kinds {
			if !(t[1] == 'm' && t[0] >= '2' && t[0] <= '8') {
				filtered = append(filtered, t)
			}
		}
		kinds = filtered
	}
	for n := byte(1); n <= 7; n++ {
		kinds = append(kinds, tileKind(n, 'z'))
	}
	deck := make([]engine.Tile, 0, len(kinds)*4)
	for _, k := range kinds {
		for i := 0; i < 4; i++ {
			deck = append(deck, k)
		}
	}
	return deck
}
