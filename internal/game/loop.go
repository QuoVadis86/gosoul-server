package game

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/qy-info/gosoul/internal/game/engine"
)

// runRound drives one engine round to completion using the coordinator.
// startRound has already dealt the dealer's extra tile; we begin asking the
// dealer's discard right away.
func (s *session) runRound(ctx context.Context) error {
	r := s.round.round
	step := uint32(1)
	err := s.playTurns(ctx, r, step, s.Seat)
	// Normal ends still get a paipu entry; errors (disconnects) do not.
	if err != nil {
		return err
	}
	return s.archive(ctx, r)
}

// archive persists the finished round as a paipu record when a store is wired,
// and credits the round-completion achievement through the session's hook.
func (s *session) archive(ctx context.Context, r *engine.Round) error {
	if s.paipu != nil {
		payload := roundSnapshot{r.Winner, r.Scores, r.Kyoku, r.Honba}
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		if err := s.paipu.Save(ctx, s.GameUUID, string(b)); err != nil {
			return err
		}
	}
	if s.ach == nil {
		return nil
	}
	return s.ach.Increment(ctx, int64(s.AccountID), PlayedGame, 1)
}

// roundSnapshot is the persisted shape of one finished round.
type roundSnapshot struct {
	Winner int
	Scores []int
	Kyoku  int
	Honba  int
}

// playTurns repeatedly: current seat draws (or continues), decides to act, and
// hands the turn around; ends on wall exhaustion or a win.
func (s *session) playTurns(ctx context.Context, r *engine.Round, step uint32, firstSeat int) error {
	seat := firstSeat
	first := true
	for r.LeftWall() > 0 {
		if !first {
			drawn, err := r.Draw(ctx)
			if err != nil {
				return err
			}
			if err := s.round.drv.Draw(ctx, r, seat, drawn, step); err != nil {
				return err
			}
			step++
		}
		first = false

		d, err := s.decidePlay(ctx, r, seat)
		if err != nil {
			return err
		}
		ended, nextSeat, err := s.applyPlay(ctx, r, seat, d, step)
		if err != nil {
			return err
		}
		if ended {
			return nil
		}
		step++
		seat = nextSeat
	}
	r.End(-1)
	return s.round.drv.LiuJu(ctx, step)
}

// Client self-operation types (GameSelfOperation.type).
const (
	opDiscard uint32 = iota
	opRiichi
	opPon
	opChi
	opKakan
	opAnkan
	opMinkan
	opTsumo = 13
)

// decide returns what the seat owner wants to do this turn.
type decide struct {
	opType uint32
	tile   engine.Tile
}

// decidePlay asks the seat owner for their play: humans via their input
// channel, bots via the AI decision maker. It maps riichi/tsumo intents so the
// loop can act on them.
func (s *session) decidePlay(ctx context.Context, r *engine.Round, seat int) (decide, error) {
	robot := s.round.drv.BotFor(seat)
	if robot != nil {
		v := r.ViewFor(seat)
		// Bots may choose riichi or tsumo via their self-action hook.
		if sel := robot.ChooseSelfAction(ctx, v, selfOps()); sel != nil {
			if *sel == engine.OpTsumo {
				return decide{opType: opTsumo, tile: engine.Tile("")}, nil
			}
			if *sel == engine.OpRiichi {
				return decide{opType: opRiichi, tile: robot.ChooseDiscard(ctx, v)}, nil
			}
		}
		tile := robot.ChooseDiscard(ctx, v)
		if tile == "" && len(v.Hand) == 0 {
			return decide{}, fmt.Errorf("bot seat %d has empty hand", seat)
		}
		return decide{opType: opDiscard, tile: tile}, nil
	}
	ch := s.round.drv.HumanIn(seat)
	select {
	case op := <-ch:
		return decide{opType: op.Type, tile: op.Tile}, nil
	case <-ctx.Done():
		return decide{}, ctx.Err()
	}
}

func selfOps() []engine.SelfOp {
	return []engine.SelfOp{engine.OpDiscard, engine.OpRiichi, engine.OpTsumo}
}

// applyPlay executes the seat's chosen action on the engine round.
func (s *session) applyPlay(ctx context.Context, r *engine.Round, seat int, d decide, step uint32) (ended bool, nextSeat int, err error) {
	switch d.opType {
	case opRiichi:
		err = r.DeclareRiichi(ctx, d.tile)
		if err == nil {
			err = s.round.drv.Discard(ctx, r, seat, d.tile, step)
		}
		return false, (seat + 1) % r.Meta.NumPlayers, err
	case opTsumo:
		if w := engine.CheckWin(seat, r.ViewFor(seat).Hand, r.Players[seat].Melds, true, -1, r.Dora, r.IsDoubleRiichi(seat)); w != nil {
			r.End(w.Seat)
			return true, seat, s.round.drv.Tsumo(ctx, w, step)
		}
		// Bad tsumo intent without a win: fall back to a discard.
		v := r.ViewFor(seat)
		d.tile = firstHandTile(v)
		fallthrough
	default:
		if err := r.Discard(ctx, d.tile); err != nil {
			return false, seat, fmt.Errorf("discard %q type=%d: %w", d.tile, d.opType, err)
		}
		return false, (seat + 1) % r.Meta.NumPlayers, s.round.drv.Discard(ctx, r, seat, d.tile, step)
	}
}

func firstHandTile(v *engine.View) engine.Tile {
	if len(v.Hand) > 0 {
		return v.Hand[0]
	}
	return ""
}
