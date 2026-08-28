// Package ai defines the decision-making abstraction for bot seats.
//
// Design goals:
//   - The Player interface is pure: no I/O, no engine mutation. A Player only
//     answers "what would I do in this state?" from an immutable engine.View.
//   - Difficulty is pluggable. Built-in levels (Novice/Normal/Expert) ship with
//     the server; stronger or custom models register themselves by name and can
//     be selected per seat in the lobby/game config.
//   - External models (Mortal, Akochan, self-trained nets) attach through the
//     mjai JSONL-subprocess adapter, which is the de-facto standard protocol in
//     the riichi AI ecosystem.
package ai

import (
	"context"
	"fmt"
	"sync"

	"github.com/qy-info/gosoul/internal/game/engine"
)

// Player is the decision interface every bot seat implements.
//
// Implementations must be safe for concurrent use only if documented; the game
// server serializes decisions for a given seat, so a Player can hold mutable
// state as long as it is never shared across seats.
type Player interface {
	// Name identifies the player (used in room display and logs).
	Name() string

	// Level reports the configured strength tier.
	Level() Level

	// ChooseDiscard returns a tile from Hand to discard after a draw.
	ChooseDiscard(ctx context.Context, v *engine.View) engine.Tile

	// ChooseCall decides what to do when presented with meld/ron candidates
	// after an opponent's discard. Return nil to pass.
	ChooseCall(ctx context.Context, v *engine.View, ops []engine.CallOp) *engine.CallOp

	// ChooseSelfAction decides the action on the player's own turn when
	// multiple self-ops (riichi/ankan/kakan/tsumo/discard) are available.
	// Return nil to discard normally.
	ChooseSelfAction(ctx context.Context, v *engine.View, ops []engine.SelfOp) *engine.SelfOp
}

// Level is a strength tier descriptor.
type Level int

const (
	LevelNovice   Level = iota // random-ish, minimal tile efficiency
	LevelNormal                // tile-efficiency heuristic with basic defense
	LevelExpert                // full hand analysis (port of the reference engine)
	LevelExternal              // any registered custom/external model
	LevelCount
)

func (l Level) String() string {
	switch l {
	case LevelNovice:
		return "novice"
	case LevelNormal:
		return "normal"
	case LevelExpert:
		return "expert"
	case LevelExternal:
		return "external"
	default:
		return fmt.Sprintf("level(%d)", int(l))
	}
}

// Factory builds a Player. Config is interpreted per implementation.
type Factory func(cfg map[string]string) (Player, error)

var (
	registryMu sync.RWMutex
	registry   = map[string]Factory{}
)

// Register adds a Player factory under name. Built-ins and external adapters
// call this at init time; server config references players by name.
func Register(name string, f Factory) {
	if f == nil {
		panic("ai: nil factory")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry[name]; dup {
		panic("ai: factory already registered: " + name)
	}
	registry[name] = f
}

// New instantiates a player by registered name with the given config.
func New(name string, cfg map[string]string) (Player, error) {
	registryMu.RLock()
	f, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("ai: unknown player %q (registered: %v)", name, registeredNames())
	}
	return f(cfg)
}

// Names lists all registered player factories.
func Names() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registeredNames()
}

func registeredNames() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	return names
}
