// Package game owns the in-game FastTest RPC surface: seat authentication,
// game entry/sync, and (as the engine lands) round pushdown via action
// notifies. Handlers live here so room→auth→round wiring stays in one layer.
package game

import (
	"log/slog"
	"time"

	"github.com/qy-info/gosoul/internal/paipu"
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
	drv       *drive
	paipu     *paipu.Service
	ach       Achievements
	kyoku     int
	honba     int
	// numPlayers is the table size (3 for sanma, 4 for yonma).
	numPlayers int
}

// Handlers registers the FastTest surface on r.
func Handlers(r *router.Router, log *slog.Logger, pp *paipu.Service, ach Achievements) {
	if ach == nil {
		ach = noopAchievements{}
	}
	h := &handlers{log: log, sessions: make(map[router.Session]*session), paipu: pp, ach: ach}
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
	paipu    *paipu.Service
	ach      Achievements
}

func (h *handlers) ok(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResCommon, &resCommon{Error: &errBody{}})
}
