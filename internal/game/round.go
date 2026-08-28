package game

import (
	"context"
	"sort"

	"github.com/qy-info/gosoul/internal/deal"
	"github.com/qy-info/gosoul/internal/game/ai"
	"github.com/qy-info/gosoul/internal/game/engine"
	"github.com/qy-info/gosoul/internal/protocol"
	"github.com/qy-info/gosoul/internal/router"
)

// roundState is the live engine round attached to a seat session.
type roundState struct {
	round *engine.Round
	drv   *drive
	// kyoku/honba track the match position (engine owns per-round state).
	kyoku int
	honba int
}

// startRound builds a fresh wall, constructs the engine round, and pushes the
// ActionNewRound notify to the seat so the client renders the initial hand.
func (h *handlers) startRound(ctx context.Context, s *session, sess router.Session) error {
	factory := deal.RandomWallFactory{}
	n := s.numPlayers
	if n == 0 {
		n = 4
	}
	meta := engine.RoundMeta{
		NumPlayers:   n,
		InitialScore: 25000,
		Kyoku:        s.kyoku,
		Honba:        s.honba,
		Sanma:        n == 3,
		NotenBappu:   true,
	}
	w, err := factory.BuildWall(ctx, meta)
	if err != nil {
		return err
	}
	// The session waits on the human's input for its own turns, so the engine
	// runs with a passthrough driver that the human input handler replaces
	// seat-by-seat. Bots decide through the same driver once wired to ai.
	driver := &noopDriver{}
	r := engine.NewRound(meta, w, driver)
	if _, err := r.Start(ctx); err != nil {
		return err
	}
	drv := newDrive(sess, h.log, botForSeats())
	s.round = &roundState{round: r, drv: drv, kyoku: s.kyoku, honba: s.honba}

	human := r.ViewFor(s.Seat)
	r.SortHand(s.Seat)

	action := &actionNewRound{
		Chang:         uint32(r.Kyoku / r.Meta.NumPlayers),
		Ju:            uint32(r.Kyoku % r.Meta.NumPlayers),
		Ben:           uint32(r.Honba),
		Liqibang:      uint32(r.RiichiStick),
		Dora:          string(r.Dora[0]),
		Doras:         tileStrings(r.Dora),
		Scores:        append([]int(nil), r.Scores...),
		Tiles:         tileStrings(human.Hand),
		LeftTileCount: uint32(r.LeftWall()),
	}
	if err := sess.ActionNotify(protocol.ActionNewRound, action, 0); err != nil {
		return err
	}
	h.log.Info("round started", "game", s.GameUUID, "seat", s.Seat, "dora", string(r.Dora[0]))
	go func() {
		if err := s.runRound(context.Background()); err != nil {
			h.log.Warn("round drive ended", "err", err)
		}
	}()
	return nil
}

// noopDriver is a placeholder decision driver for the mixed human/AI round;
// per-seat decisions arrive as inputOperation calls and are routed by the
// session handler.
type noopDriver struct{}

func (noopDriver) DiscardTile(context.Context, *engine.View) engine.Tile { return "" }
func (noopDriver) DeclareRiichi(context.Context, *engine.View) bool      { return false }

// actionNewRound mirrors the ActionNewRound payload.
type actionNewRound struct {
	Chang         uint32   `json:"chang"`
	Ju            uint32   `json:"ju"`
	Ben           uint32   `json:"ben"`
	Liqibang      uint32   `json:"liqibang"`
	Dora          string   `json:"dora"`
	Doras         []string `json:"doras"`
	Scores        []int    `json:"scores"`
	Tiles         []string `json:"tiles"`
	LeftTileCount uint32   `json:"leftTileCount"`
}

func tileStrings(tiles []engine.Tile) []string {
	out := make([]string, 0, len(tiles))
	for _, t := range tiles {
		out = append(out, string(t))
	}
	sort.Strings(out)
	return out
}

// botForSeats returns a bot provider: every seat gets a normal bot except seat
// 0, which is the human's. Real seat mapping comes from room config later.
func botForSeats() func(seat int) ai.Player {
	return func(seat int) ai.Player {
		if seat == 0 {
			return nil
		}
		p, err := ai.New("normal", nil)
		if err != nil {
			return nil
		}
		return p
	}
}
