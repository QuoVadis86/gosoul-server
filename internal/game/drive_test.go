package game

import (
	"context"
	"sync"
	"testing"

	"github.com/qy-info/gosoul/internal/deal"
	"github.com/qy-info/gosoul/internal/game/ai"
	"github.com/qy-info/gosoul/internal/game/engine"
	"github.com/qy-info/gosoul/internal/paipu"
	"github.com/qy-info/gosoul/internal/storage"
)

// fakeSess records actions pushed through ActionNotify.
type fakeSess struct {
	mu      sync.Mutex
	actions []string
}

func (f *fakeSess) ActionNotify(action string, v any, step uint32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.actions = append(f.actions, action)
	return nil
}
func (f *fakeSess) AccountID() int64                        { return 1 }
func (f *fakeSess) SetAccountID(id int64)                   {}
func (f *fakeSess) Respond(_ uint16, _ string, _ any) error { return nil }
func (f *fakeSess) Notify(_ string, _ any) error            { return nil }

// BotFor drives every seat by a fixed bot; used to prove the whole round
// advances without human input.
type fixedBot struct{ tile engine.Tile }

func (b *fixedBot) Name() string    { return "fixed" }
func (b *fixedBot) Level() ai.Level { return ai.LevelNormal }
func (b *fixedBot) ChooseDiscard(_ context.Context, v *engine.View) engine.Tile {
	if b.tile != "" {
		return b.tile
	}
	if len(v.Hand) > 0 {
		return v.Hand[0]
	}
	return ""
}
func (b *fixedBot) ChooseCall(context.Context, *engine.View, []engine.CallOp) *engine.CallOp {
	return nil
}
func (b *fixedBot) ChooseSelfAction(context.Context, *engine.View, []engine.SelfOp) *engine.SelfOp {
	return nil
}
func (b *fixedBot) DeclareRiichi(context.Context, *engine.View) bool { return false }
func (b *fixedBot) DiscardTile(_ context.Context, v *engine.View) engine.Tile {
	return b.ChooseDiscard(context.Background(), v)
}

func TestDriveRunsToWallExhaustion(t *testing.T) {
	fake := &fakeSess{}
	s := &session{
		Seat:  0,
		round: &roundState{},
	}
	wall := &engine.Wall{
		Hands:       make([][]engine.Tile, 4),
		DeadWall:    []engine.Tile{"1m", "2m", "3z", "4z", "5z", "6z", "7z", "1p", "2p", "3p", "4p", "5p", "6p", "7p"},
		Wall:        nil,
		DealerExtra: "5z",
	}
	for i := range wall.Hands {
		wall.Hands[i] = []engine.Tile{"1m", "2m", "3m", "4m", "5m", "6m", "7m", "8m", "9m", "1p", "1s", "1z", "2z"}
	}
	for i := 0; i < 20; i++ {
		wall.Wall = append(wall.Wall, []engine.Tile{"1m", "2m", "3p", "4s", "5z"}...)
	}
	d := &fixedBot{tile: ""} // discard first tile each turn
	drv := newDrive(fake, nil, func(seat int) ai.Player {
		return d // all four seats auto-play; no human input needed
	})
	r := engine.NewRound(engine.RoundMeta{NumPlayers: 4, InitialScore: 25000, Kyoku: 0}, wall, d)
	s.round.round = r
	s.round.drv = drv

	// Dealer draws the extra tile.
	if _, err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := s.runRound(context.Background()); err != nil {
		t.Fatalf("runRound: %v", err)
	}
	fake.mu.Lock()
	got := len(fake.actions)
	fake.mu.Unlock()
	if got == 0 {
		t.Fatal("no actions pushed")
	}
	if got < 5 {
		t.Fatalf("expected several actions, got %d", got)
	}
}

func TestDriveHumanInputRequired(t *testing.T) {
	fake := &fakeSess{}
	s := &session{Seat: 0, round: &roundState{}}
	wall := &engine.Wall{
		Hands:       make([][]engine.Tile, 4),
		DeadWall:    []engine.Tile{"1m", "2m", "3z", "4z", "5z", "6z", "7z", "1p", "2p", "3p", "4p", "5p", "6p", "7p"},
		Wall:        []engine.Tile{},
		DealerExtra: "5z",
	}
	for i := range wall.Hands {
		wall.Hands[i] = []engine.Tile{"1m", "2m", "3m", "4m", "5m", "6m", "7m", "8m", "9m", "1p", "1s", "1z", "2z"}
	}
	d := &fixedBot{tile: "1m"}
	drv := newDrive(fake, nil, func(seat int) ai.Player {
		if seat == 0 {
			return nil // human
		}
		return d
	})
	r := engine.NewRound(engine.RoundMeta{NumPlayers: 4, InitialScore: 25000, Kyoku: 0}, wall, d)
	s.round.round = r
	s.round.drv = drv
	if _, err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	ch := drv.HumanIn(0)
	done := make(chan error, 1)
	go func() {
		done <- s.runRound(context.Background())
	}()
	// Without human input the goroutine blocks on HumanIn; deliver after a beat.
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected round to wait for human input")
		}
		return
	default:
	}
	ch <- humanOp{Tile: "1m"}
	if err := <-done; err != nil {
		t.Fatalf("round failed after input: %v", err)
	}
}

func TestArchiveWritesPaipu(t *testing.T) {
	fake := &fakeSess{}
	store, err := newTempStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	s := &session{
		Seat:     0,
		GameUUID: "g-archive-1",
		round:    &roundState{},
		paipu:    paipu.New(store.Paipu),
	}
	wall := &engine.Wall{
		Hands:       make([][]engine.Tile, 4),
		DeadWall:    []engine.Tile{"1m", "2m", "3z", "4z", "5z", "6z", "7z", "1p", "2p", "3p", "4p", "5p", "6p", "7p"},
		Wall:        []engine.Tile{},
		DealerExtra: "5z",
	}
	for i := range wall.Hands {
		wall.Hands[i] = []engine.Tile{"1m", "2m", "3m", "4m", "5m", "6m", "7m", "8m", "9m", "1p", "1s", "1z", "2z"}
	}
	for i := 0; i < 20; i++ {
		wall.Wall = append(wall.Wall, []engine.Tile{"1m", "2m", "3p", "4s", "5z"}...)
	}
	d := &fixedBot{tile: ""}
	r := engine.NewRound(engine.RoundMeta{NumPlayers: 4, InitialScore: 25000, Kyoku: 0}, wall, d)
	s.round.round = r
	s.round.drv = newDrive(fake, nil, func(int) ai.Player { return d })
	if _, err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := s.runRound(context.Background()); err != nil {
		t.Fatalf("runRound: %v", err)
	}
	rec, err := s.paipu.Get(context.Background(), "g-archive-1")
	if err != nil {
		t.Fatalf("paipu get: %v", err)
	}
	if rec.JSON == "" || rec.UUID != "g-archive-1" {
		t.Fatalf("bad paipu: %+v", rec)
	}
}

func newTempStore(t *testing.T) (*storage.Store, error) {
	t.Helper()
	return storage.Open(t.TempDir() + "/db.sqlite")
}

type fakeAch struct {
	mu        sync.Mutex
	accountID int64
	achieveID int64
	delta     int64
	calls     int
}

func (f *fakeAch) Increment(_ context.Context, accountID, achieveID, delta int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.accountID, f.achieveID, f.delta = accountID, achieveID, delta
	f.calls++
	return nil
}

func TestArchiveCreditsAchievement(t *testing.T) {
	store, err := newTempStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	fake := &fakeSess{}
	fa := &fakeAch{}
	s := &session{
		AccountID: 9,
		Seat:      0,
		GameUUID:  "g-ach-1",
		round:     &roundState{},
		paipu:     paipu.New(store.Paipu),
		ach:       fa,
	}
	wall := &engine.Wall{
		Hands:       make([][]engine.Tile, 4),
		DeadWall:    []engine.Tile{"1m", "2m", "3z", "4z", "5z", "6z", "7z", "1p", "2p", "3p", "4p", "5p", "6p", "7p"},
		Wall:        nil,
		DealerExtra: "5z",
	}
	for i := range wall.Hands {
		wall.Hands[i] = []engine.Tile{"1m", "2m", "3m", "4m", "5m", "6m", "7m", "8m", "9m", "1p", "1s", "1z", "2z"}
	}
	for i := 0; i < 20; i++ {
		wall.Wall = append(wall.Wall, []engine.Tile{"1m", "2m", "3p", "4s", "5z"}...)
	}
	d := &fixedBot{tile: ""}
	r := engine.NewRound(engine.RoundMeta{NumPlayers: 4, InitialScore: 25000, Kyoku: 0}, wall, d)
	s.round.round = r
	s.round.drv = newDrive(fake, nil, func(int) ai.Player { return d })
	if _, err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := s.runRound(context.Background()); err != nil {
		t.Fatalf("runRound: %v", err)
	}
	fa.mu.Lock()
	defer fa.mu.Unlock()
	if fa.calls != 1 {
		t.Fatalf("achievement calls = %d, want 1", fa.calls)
	}
	if fa.accountID != 9 || fa.achieveID != PlayedGame || fa.delta != 1 {
		t.Fatalf("hook = (%d,%d,%d)", fa.accountID, fa.achieveID, fa.delta)
	}
}

func TestSanmaRoundRuns(t *testing.T) {
	fake := &fakeSess{}
	s := &session{
		Seat:       0,
		GameUUID:   "g-sanma-1",
		numPlayers: 3,
		round:      &roundState{},
	}
	factory := &deal.RandomWallFactory{}
	meta := engine.RoundMeta{NumPlayers: 3, InitialScore: 35000, Kyoku: 0, Sanma: true, NotenBappu: true}
	w, err := factory.BuildWall(context.Background(), meta)
	if err != nil {
		t.Fatal(err)
	}
	// 3 hands of 13 = 39 tiles, plus dealer extra.
	if len(w.Hands) != 3 {
		t.Fatalf("sanma hands = %d, want 3", len(w.Hands))
	}
	for i, h := range w.Hands {
		if len(h) != 13 {
			t.Fatalf("seat %d hand = %d tiles, want 13", i, len(h))
		}
		for _, tile := range h {
			// Sanma drops 2m..8m.
			if tile[1] == 'm' && tile[0] >= '2' && tile[0] <= '8' {
				t.Fatalf("sanma deck contains %s", tile)
			}
		}
	}
	r := engine.NewRound(meta, w, &fixedBot{tile: ""})
	if _, err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.Meta.NumPlayers != 3 {
		t.Fatalf("round players = %d, want 3", r.Meta.NumPlayers)
	}
	if len(r.Dora) == 0 {
		t.Fatal("sanma round missing dora indicators")
	}
	_ = s
	_ = fake
}

func TestHumanRiichiOpApplied(t *testing.T) {
	fake := &fakeSess{}
	s := &session{
		Seat:  0,
		round: &roundState{},
	}
	wall := &engine.Wall{
		Hands:       make([][]engine.Tile, 4),
		DeadWall:    []engine.Tile{"1m", "2m", "3z", "4z", "5z", "6z", "7z", "1p", "2p", "3p", "4p", "5p", "6p", "7p"},
		DealerExtra: "5z",
		Wall:        []engine.Tile{"6p"},
	}
	for i := range wall.Hands {
		wall.Hands[i] = []engine.Tile{"1m", "2m", "3m", "4m", "5m", "6m", "7m", "8m", "9m", "1p", "1s", "1z", "2z"}
	}
	d := &fixedBot{tile: ""}
	r := engine.NewRound(engine.RoundMeta{NumPlayers: 4, InitialScore: 25000, Kyoku: 0}, wall, d)
	s.round.round = r
	s.round.drv = newDrive(fake, nil, func(seat int) ai.Player {
		if seat == 0 {
			return nil // human
		}
		return d
	})
	if _, err := r.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Human declares riichi on the very first discard. With an empty wall the
	// round ends right after, keeping the test deterministic.
	ch := s.round.drv.HumanIn(0)
	done := make(chan error, 1)
	go func() {
		done <- s.runRound(context.Background())
	}()
	ch <- humanOp{Type: opRiichi, Tile: "1m"}
	if err := <-done; err != nil {
		t.Fatalf("round after riichi: %v", err)
	}
	if !r.IsDoubleRiichi(0) {
		t.Fatal("human first-discard riichi should be double")
	}
}
