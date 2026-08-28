package game

import (
	"context"

	"github.com/qy-info/gosoul/internal/game/engine"
)

// runRound drives one engine round to completion using the coordinator.
// startRound has already dealt the dealer's extra tile; we begin asking the
// dealer's discard right away.
func (s *session) runRound(ctx context.Context) error {
	r := s.round.round
	step := uint32(1)
	return s.playTurns(ctx, r, step, s.Seat)
}

// playTurns repeatedly: current seat draws (or continues), decides to discard,
// and hands the turn around; ends on wall exhaustion or a win.
func (s *session) playTurns(ctx context.Context, r *engine.Round, step uint32, firstSeat int) error {
	seat := firstSeat
	for r.LeftWall() > 0 {
		// The first turn's tile was already drawn by runRound/Start.
		if seat != firstSeat {
			drawn, err := r.Draw(ctx)
			if err != nil {
				return err
			}
			if err := s.round.drv.Draw(ctx, r, seat, drawn, step); err != nil {
				return err
			}
			step++
		}

		// Decide the discard (human or bot).
		tile, err := s.decideDiscard(ctx, r, seat)
		if err != nil {
			return err
		}
		// Consult wins before applying the discard (tsumo on the drawn tile).
		if w := engine.CheckWin(seat, r.ViewFor(seat).Hand, r.Players[seat].Melds, true, -1); w != nil {
			if err := s.round.drv.Tsumo(ctx, w, step); err != nil {
				return err
			}
			return nil
		}
		if err := r.Discard(ctx, tile); err != nil {
			return err
		}
		if err := s.round.drv.Discard(ctx, r, seat, tile, step); err != nil {
			return err
		}
		step++
		seat = (seat + 1) % r.Meta.NumPlayers
	}
	return s.round.drv.LiuJu(ctx, step)
}

// decideDiscard asks the seat owner for their play: humans via their input
// channel, bots via the AI decision maker.
func (s *session) decideDiscard(ctx context.Context, r *engine.Round, seat int) (engine.Tile, error) {
	robot := s.round.drv.BotFor(seat)
	if robot != nil {
		v := r.ViewFor(seat)
		return robot.ChooseDiscard(ctx, v), nil
	}
	ch := s.round.drv.HumanIn(seat)
	select {
	case op := <-ch:
		return op.Tile, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
