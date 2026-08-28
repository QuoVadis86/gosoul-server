// Package game owns the in-game FastTest RPC surface: seat authentication,
// game entry/sync, and (as the engine lands) round pushdown via action
// notifies. Handlers live here so room→auth→round wiring stays in one layer.
package game

import (
	"log/slog"
	"time"

	"github.com/qy-info/gosoul/internal/protocol"
	"github.com/qy-info/gosoul/internal/router"
)

// session carries one authenticated game seat's state.
type session struct {
	AccountID uint32
	Seat      int
	GameUUID  string
	StartedAt time.Time
	round     *roundState
	kyoku     int
	honba     int
}

// Handlers registers the FastTest surface on r.
func Handlers(r *router.Router, log *slog.Logger) {
	h := &handlers{log: log, sessions: make(map[router.Session]*session)}
	r.Handle(protocol.MethodFastTestAuthGame, h.authGame)
	r.Handle(protocol.MethodFastTestEnterGame, h.enterGame)
	r.Handle(protocol.MethodFastTestSyncGame, h.syncGame)
	r.Handle(protocol.MethodFastTestConfirmNewRound, h.confirmNewRound)
	r.Handle(protocol.MethodFastTestInputOperation, h.inputOperation)
	r.Handle(protocol.MethodFastTestInputChiPengGang, h.inputCPG)
	r.Handle(protocol.MethodFastTestCheckNetworkDelay, h.ok)
	r.Handle(protocol.MethodFastTestFinishSyncGame, h.ok)
	r.Handle(protocol.MethodFastTestTerminateGame, h.ok)
}

type handlers struct {
	log      *slog.Logger
	sessions map[router.Session]*session
}

func (h *handlers) ok(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResCommon, &resCommon{Error: &errBody{}})
}
