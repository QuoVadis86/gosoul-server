package deal

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"github.com/qy-info/gosoul/internal/game/engine"
)

// Preset is one named deal configuration managed by the GM console.
//
// Every field is optional:
//   - Hands pins the 13-tile deal for the given seats (index = seat).
//   - DealerExtra pins the dealer's 14th tile.
//   - WallPrefix pins the first N draws of the draw stack.
//   - DoraIndicators pins the initial dora indicator tiles (index order).
//
// Unpinned parts are filled from a shuffled remainder of the full deck, so a
// preset stays legal even when only a fragment of the deal is prescribed.
type Preset struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Hands          [][]string `json:"hands,omitempty"`
	DealerExtra    string     `json:"dealer_extra,omitempty"`
	WallPrefix     []string   `json:"wall_prefix,omitempty"`
	DoraIndicators []string   `json:"dora_indicators,omitempty"`
}

// Store is the GM-managed preset registry. It is the integration point the
// admin console writes to; the game reads through WallFactory.
type Store struct {
	mu      sync.RWMutex
	presets map[string]*Preset
}

// NewStore returns an empty preset store.
func NewStore() *Store {
	return &Store{presets: make(map[string]*Preset)}
}

// Upsert registers or replaces a preset.
func (s *Store) Upsert(p *Preset) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.presets == nil {
		s.presets = make(map[string]*Preset)
	}
	if p.ID == "" {
		p.ID = fmt.Sprintf("preset-%d", len(s.presets)+1)
	}
	s.presets[p.ID] = p
}

// Get returns a preset by ID.
func (s *Store) Get(id string) (*Preset, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.presets[id]
	return p, ok
}

func (s *Store) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.presets, id)
}

// List returns all presets.
func (s *Store) List() []*Preset {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Preset, 0, len(s.presets))
	for _, p := range s.presets {
		out = append(out, p)
	}
	return out
}

// Name returns a wall factory that turns the preset ID into a deal. If the
// preset is missing, the factory falls back to random dealing.
func (s *Store) Name(id string) engine.WallFactory {
	return &presetFactory{store: s, id: id}
}

type presetFactory struct {
	store *Store
	id    string
}

func (f *presetFactory) BuildWall(ctx context.Context, meta engine.RoundMeta) (*engine.Wall, error) {
	p, ok := f.store.Get(f.id)
	if !ok || p == nil {
		return RandomWallFactory{}.BuildWall(ctx, meta)
	}

	base := buildShuffledWall(meta)
	w := &engine.Wall{}

	// Prescribed seat hands, converted to canonical tiles.
	for i := 0; i < meta.NumPlayers; i++ {
		if i < len(p.Hands) && len(p.Hands[i]) > 0 {
			hand := make([]engine.Tile, 0, len(p.Hands[i]))
			for _, t := range p.Hands[i] {
				hand = append(hand, engine.Tile(t))
			}
			w.Hands = append(w.Hands, hand)
		} else if i < len(base.Hands) {
			w.Hands = append(w.Hands, base.Hands[i])
		}
	}

	if p.DealerExtra != "" {
		w.DealerExtra = engine.Tile(p.DealerExtra)
	} else {
		w.DealerExtra = base.DealerExtra
	}

	if len(p.WallPrefix) > 0 {
		prefix := make([]engine.Tile, 0, len(p.WallPrefix))
		for _, t := range p.WallPrefix {
			prefix = append(prefix, engine.Tile(t))
		}
		w.Wall = append(prefix, base.Wall[len(prefix):]...)
	} else {
		w.Wall = base.Wall
	}

	if len(p.DoraIndicators) > 0 {
		dora := make([]engine.Tile, 0, len(p.DoraIndicators))
		for _, t := range p.DoraIndicators {
			dora = append(dora, engine.Tile(t))
		}
		// Rebuild the dead wall around the prescribed indicators while leaving
		// room for kan reveals; pad the rest from the base dead wall.
		newDead := make([]engine.Tile, len(base.DeadWall))
		for i, t := range dora {
			if 4+i*2 < len(newDead) {
				newDead[4+i*2] = t
			}
		}
		rand.Shuffle(len(newDead), func(a, b int) {
			if newDead[a] != "" && newDead[b] != "" {
				newDead[a], newDead[b] = newDead[b], newDead[a]
			}
		})
		for i := range newDead {
			if newDead[i] == "" {
				newDead[i] = base.DeadWall[i]
			}
		}
		w.DeadWall = newDead
	} else {
		w.DeadWall = base.DeadWall
	}

	return w, nil
}
