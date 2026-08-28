package game

import (
	"context"
	"errors"
	"log/slog"

	"github.com/qy-info/gosoul/internal/game/ai"
	"github.com/qy-info/gosoul/internal/game/engine"
	"github.com/qy-info/gosoul/internal/protocol"
	"github.com/qy-info/gosoul/internal/router"
)

// humanOp carries one decoded inputOperation from the client.
type humanOp struct {
	Type      uint32
	Tile      engine.Tile
	Cancel    bool
	TimeUse   uint32
	TileState int32
}

// ErrDriveDone is the terminal result of a round drive.
var ErrDriveDone = errors.New("game: drive completed")

// drive is the per-session turn coordinator: it consumes draws/discards,
// consults human input or AI bots, and pushes action notifies in order.
type drive struct {
	sess   router.Session
	log    *slog.Logger
	botFor func(seat int) ai.Player
	humans map[int]chan humanOp
}

func newDrive(sess router.Session, log *slog.Logger, botFor func(int) ai.Player) *drive {
	return &drive{sess: sess, log: log, botFor: botFor, humans: make(map[int]chan humanOp)}
}

// BotFor returns the AI for a seat, or nil when the seat is human-driven.
func (d *drive) BotFor(seat int) ai.Player {
	if d.botFor == nil {
		return nil
	}
	return d.botFor(seat)
}

// registerDest flags a seat as human-driven and returns its input channel.
func (d *drive) registerDest(seat int) chan humanOp {
	ch := make(chan humanOp, 1)
	d.humans[seat] = ch
	return ch
}

// HumanIn returns the input channel for a human seat.
func (d *drive) HumanIn(seat int) chan humanOp {
	ch, ok := d.humans[seat]
	if !ok {
		ch = d.registerDest(seat)
	}
	return ch
}

// Deliver is called by the inputOperation handler with a human's action.
func (d *drive) Deliver(seat int, op humanOp) error {
	ch, ok := d.humans[seat]
	if !ok {
		return ErrDriveDone
	}
	ch <- op
	return nil
}

// Draw is a thin wrapper turning engine draw + broadcast into a step.
func (d *drive) Draw(ctx context.Context, r *engine.Round, seat int, drawn engine.Tile, step uint32) error {
	action := &actionDealTile{
		Seat:          uint32(seat),
		Tile:          string(drawn),
		LeftTileCount: uint32(r.LeftWall()),
		Doras:         tileStrings(r.Dora),
	}
	return d.push(protocol.ActionDealTile, action, step)
}

// Discard reports a discard (human or bot) to clients.
func (d *drive) Discard(ctx context.Context, r *engine.Round, seat int, tile engine.Tile, step uint32) error {
	action := &actionDiscardTile{
		Seat:     uint32(seat),
		Tile:     string(tile),
		Doras:    tileStrings(r.Dora),
		Scores:   append([]int(nil), r.Scores...),
		Liqibang: uint32(r.RiichiStick),
	}
	return d.push(protocol.ActionDiscardTile, action, step)
}

// Tsumo announces a win to every seat.
func (d *drive) Tsumo(ctx context.Context, win *engine.Win, step uint32) error {
	action := &actionHu{
		Hules: []huInfo{{
			Seat:  uint32(win.Seat),
			Zimo:  win.Tsumo,
			Count: uint32(win.Han),
			Fu:    uint32(win.Fu),
			Title: joinYaku(win.Yaku),
			Doras: []string{},
		}},
	}
	return d.push(protocol.ActionHule, action, step)
}

// LiuJu announces noten exhaustion.
func (d *drive) LiuJu(ctx context.Context, step uint32) error {
	return d.push(protocol.ActionLiuJu, &actionLiuJu{Type: 1}, step)
}

// CPG broadcasts a claimed meld (pon/daiminkan) to the client.
func (d *drive) CPG(ctx context.Context, r *engine.Round, seat int, meld engine.Meld, step uint32) error {
	tiles := make([]string, 0, len(meld.Tiles))
	for _, t := range meld.Tiles {
		tiles = append(tiles, string(t))
	}
	action := &actionCPG{
		Seat:  uint32(seat),
		Type:  uint32(meld.Type),
		Tiles: tiles,
	}
	return d.push(protocol.ActionChiPengGang, action, step)
}

func (d *drive) push(name string, v any, step uint32) error {
	return d.sess.ActionNotify(name, v, step)
}

func joinYaku(y []string) string {
	out := ""
	for i, s := range y {
		if i > 0 {
			out += "/"
		}
		out += s
	}
	return out
}
