package lobby

import (
	"log/slog"
	"strings"

	"github.com/qy-info/gosoul/internal/protocol"
	"github.com/qy-info/gosoul/internal/router"
)

// registerEmptySurface fills the full Majsoul RPC surface with correct-typed
// empty responses. Every method the client may call — rooms, matching, mail,
// shop, achievements, contests, battle (FastTest) — gets a registered handler
// that answers its real proto response type with an empty success payload.
//
// The point of this layer is protocol completeness: the client never hangs on
// an unhandled method and every response decodes under the correct proto type.
// Real logic replaces these handlers one by one as features are implemented.
func registerEmptySurface(r *router.Router, reg *protocol.Registry, log *slog.Logger) {
	handlers := 0
	for _, method := range reg.Methods() {
		if r.Has(method) {
			continue
		}
		// Battle methods arrive over the same connection (FastTest service),
		// so they are surfaced here until the game session layer lands.
		if !isSurfaceMethod(method) {
			continue
		}
		method := method
		r.Handle(method, func(ctx *router.Context) error {
			route, ok := ctx.Reg.RouteFor(method)
			if !ok {
				return ctx.Session.Respond(ctx.MsgID, protocol.TypeResCommon, empty{})
			}
			return ctx.Session.Respond(ctx.MsgID, route.RespType, empty{})
		})
		handlers++
	}
	log.Info("lobby surface registered", "handlers", handlers)
}

func isSurfaceMethod(method string) bool {
	return strings.HasPrefix(method, protocol.LobbyServicePrefix) ||
		strings.HasPrefix(method, protocol.FastTestServicePrefix) ||
		strings.HasPrefix(method, protocol.RouteServicePrefix)
}
