// Package lobby implements the account-facing lobby RPC handlers: route
// connection, login/registration and home data. Handlers translate liqi
// payloads (DTOs) onto the lobby domain service.
package lobby

import (
	"context"
	"log/slog"

	"github.com/qy-info/gosoul/internal/protocol"
	"github.com/qy-info/gosoul/internal/router"
	"github.com/qy-info/gosoul/internal/user"
)

// Handlers registers every lobby RPC the client uses at login.
func Handlers(svc *Service, accounts *user.AccountService, chars *user.CharacterService, wallets *user.CurrencyService, log *slog.Logger, r *router.Router, reg *protocol.Registry) {
	h := &handler{svc: svc, accounts: accounts, chars: chars, wallets: wallets, log: log}

	r.Handle(".lq.Route.requestConnection", h.requestConnection)
	r.Handle(".lq.Route.heartbeat", h.heartbeat)
	r.Handle(".lq.Lobby.prepareLogin", h.ok)
	r.Handle(".lq.Lobby.login", h.login)
	r.Handle(".lq.Lobby.emailLogin", h.login)
	r.Handle(".lq.Lobby.fastLogin", h.login)
	r.Handle(".lq.Lobby.oauth2Login", h.login)
	r.Handle(".lq.Lobby.oauth2Auth", h.oauth2Auth)
	r.Handle(".lq.Lobby.oauth2Check", h.oauth2Check)
	r.Handle(".lq.Lobby.oauth2Signup", h.ok)
	r.Handle(".lq.Lobby.openidCheck", h.oauth2Check)
	r.Handle(".lq.Lobby.loginSuccess", h.loginSuccess)
	r.Handle(".lq.Lobby.loginBeat", h.ok)
	r.Handle(".lq.Lobby.logout", h.ok)
	r.Handle(".lq.Lobby.checkPrivacy", h.ok)
	r.Handle(".lq.Lobby.bindOauth2", h.ok)
	r.Handle(".lq.Lobby.fetchLastPrivacy", h.fetchLastPrivacy)
	r.Handle(".lq.Lobby.fetchInfo", h.fetchInfo)
	r.Handle(".lq.Lobby.fetchConnectionInfo", h.fetchConnectionInfo)

	registerEmptySurface(r, reg, log)
}

type handler struct {
	svc      *Service
	accounts *user.AccountService
	chars    *user.CharacterService
	wallets  *user.CurrencyService
	log      *slog.Logger
}

func (h *handler) requestConnection(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, "lq.ResRequestConnection", &resRequestConnection{
		Result:    1,
		Timestamp: uint32(now()),
	})
}

func (h *handler) heartbeat(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, "lq.ResHeartbeat", &resHeartbeat{})
}

type reqLogin struct {
	Account     string `json:"account"`
	Password    string `json:"password"`
	Email       string `json:"email"`
	AccessToken string `json:"accessToken"`
}

// login authenticates (auto-registering unknown names) and returns ResLogin.
func (h *handler) login(ctx *router.Context) error {
	var req reqLogin
	if err := ctx.Reg.DecodeInto(reqType(ctx.Method), ctx.Payload, &req); err != nil {
		return err
	}
	username := req.Account
	if username == "" {
		username = req.Email
	}
	if username == "" {
		username = "default"
	}

	state, err := h.svc.Login(context.Background(), LoginRequest{Username: username, Password: req.Password})
	if err != nil {
		h.log.Error("login failed", "user", username, "err", err)
		return err
	}
	ctx.Session.SetAccountID(state.Account.ID)

	return ctx.Session.Respond(ctx.MsgID, "lq.ResLogin", &resLogin{
		AccountID:             uint32(state.Account.ID),
		Account:               h.accountRPC(state),
		AccessToken:           accessToken(state.Account.ID),
		HasUnreadAnnouncement: false,
		Country:               "chs",
		IsIDCardAuthed:        true,
		SignupTime:            uint32(state.Account.CreatedAt),
	})
}

func (h *handler) oauth2Auth(ctx *router.Context) error {
	acc, err := h.accounts.Get(context.Background(), 1)
	if err != nil {
		return err
	}
	ctx.Session.SetAccountID(acc.ID)
	return ctx.Session.Respond(ctx.MsgID, "lq.ResOauth2Auth", &resOauth2Auth{
		AccessToken: accessToken(acc.ID),
	})
}

func (h *handler) oauth2Check(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, "lq.ResOauth2Check", &resOauth2Check{HasAccount: true})
}

// loginSuccess finalises login and pushes initial account state.
func (h *handler) loginSuccess(ctx *router.Context) error {
	if err := ctx.Session.Respond(ctx.MsgID, "lq.ResCommon", empty{}); err != nil {
		return err
	}
	return ctx.Session.Notify(".lq.NotifyAccountUpdate", &notifyAccountUpdate{
		Update: h.updatePayload(ctx.Session.AccountID()),
	})
}

func (h *handler) fetchInfo(ctx *router.Context) error {
	state, err := h.svc.Home(context.Background(), ctx.Session.AccountID())
	if err != nil {
		return err
	}
	dto := &resFetchInfo{}
	dto.ServerTime.ServerTime = now()
	dto.CharacterInfo.Characters = h.charRPCs(state)
	if len(state.Characters) > 0 {
		dto.CharacterInfo.MainCharacterID = uint32(state.Characters[0].CharID)
		dto.CharacterInfo.Skins = characterSkins(state)
	}
	return ctx.Session.Respond(ctx.MsgID, "lq.ResFetchInfo", dto)
}

func (h *handler) fetchLastPrivacy(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, "lq.ResFetchLastPrivacy", &resFetchLastPrivacy{
		Privacy: []privacyVersion{
			{Type: 1, Version: "USER-20210715-1"},
			{Type: 2, Version: "PRIVACY-20210715-1"},
		},
	})
}

func (h *handler) fetchConnectionInfo(ctx *router.Context) error {
	info := &resConnectionInfo{}
	info.ClientEndpoint.Address = "127.0.0.1"
	info.ClientEndpoint.Port = 8443
	info.ClientEndpoint.Family = "IPv4"
	return ctx.Session.Respond(ctx.MsgID, "lq.ResConnectionInfo", info)
}

func (h *handler) ok(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, "lq.ResCommon", empty{})
}

// reqType maps a login-family method to its proto request type.
func reqType(method string) string {
	switch method {
	case ".lq.Lobby.oauth2Login":
		return "lq.ReqOauth2Login"
	case ".lq.Lobby.emailLogin":
		return "lq.ReqEmailLogin"
	case ".lq.Lobby.fastLogin":
		return "lq.ReqFastLogin"
	default:
		return "lq.ReqLogin"
	}
}
