package ai

import (
	"context"
	"math/rand"

	"github.com/qy-info/gosoul/internal/game/engine"
)

func init() {
	Register("novice", func(cfg map[string]string) (Player, error) {
		return &noviceBot{baseBot: baseBot{name: "novice"}}, nil
	})
	Register("normal", func(cfg map[string]string) (Player, error) {
		return &normalBot{baseBot: baseBot{name: "normal"}}, nil
	})
	Register("expert", func(cfg map[string]string) (Player, error) {
		return &expertBot{baseBot: baseBot{name: "expert"}}, nil
	})
}

type baseBot struct{ name string }

func (b *baseBot) Name() string { return b.name }
func (b *baseBot) Level() Level { return LevelNormal }
func (b *baseBot) ChooseCall(_ context.Context, _ *engine.View, _ []engine.CallOp) *engine.CallOp {
	return nil
}

func (b *baseBot) ChooseSelfAction(_ context.Context, _ *engine.View, ops []engine.SelfOp) *engine.SelfOp {
	for _, op := range ops {
		if op == engine.OpTsumo {
			c := op
			return &c
		}
	}
	return nil
}

type noviceBot struct{ baseBot }

func (b *noviceBot) Level() Level { return LevelNovice }

func (b *noviceBot) ChooseDiscard(_ context.Context, v *engine.View) engine.Tile {
	hand := v.Hand
	if len(hand) == 0 {
		return ""
	}
	return hand[rand.Intn(len(hand))]
}

type normalBot struct{ baseBot }

func (b *normalBot) ChooseDiscard(ctx context.Context, v *engine.View) engine.Tile {
	return discardByIsolation(v)
}

type expertBot struct{ baseBot }

func (b *expertBot) Level() Level { return LevelExpert }

func (b *expertBot) ChooseDiscard(ctx context.Context, v *engine.View) engine.Tile {
	// TODO: port full shanten + tile-efficiency + defense analysis from the
	// reference engine. For now fall back to normal-tier heuristics.
	return discardByIsolation(v)
}

// discardByIsolation is a simple tile-efficiency heuristic: keep pairs and
// tiles adjacent to others; discard the most isolated tile first.
func discardByIsolation(v *engine.View) engine.Tile {
	counts := engine.TileSet{}
	for _, t := range v.Hand {
		counts[t]++
	}

	best := engine.Tile("")
	bestScore := -1
	for _, t := range v.Hand {
		s := isolationScore(t)
		if counts[t] > 1 {
			s = 10 // keep pairs
		}
		if s > bestScore {
			bestScore = s
			best = t
		}
	}
	if best == "" && len(v.Hand) > 0 {
		return v.Hand[0]
	}
	return best
}

func isolationScore(t engine.Tile) int {
	if len(t) != 2 || t[1] == 'z' {
		return 8 // terminals, honors: least useful unless pairing
	}
	n := int(t[0] - '0')
	if n == 1 || n == 9 {
		return 9
	}
	return 10 + n // inner tiles are most useful; higher = more discardable
}
