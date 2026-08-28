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
func Handlers(svc *user.Service, log *slog.Logger, r *router.Router, reg *protocol.Registry) {
	h := &handler{user: svc, log: log}

	r.Handle(protocol.MethodRouteRequestConnection, h.requestConnection)
	r.Handle(protocol.MethodRouteHeartbeat, h.heartbeat)
	r.Handle(protocol.MethodLobbyPrepareLogin, h.ok)
	r.Handle(protocol.MethodLobbyLogin, h.login)
	r.Handle(protocol.MethodLobbyEmailLogin, h.login)
	r.Handle(protocol.MethodLobbyFastLogin, h.login)
	r.Handle(protocol.MethodLobbyOauth2Login, h.login)
	r.Handle(protocol.MethodLobbyOauth2Auth, h.oauth2Auth)
	r.Handle(protocol.MethodLobbyOauth2Check, h.oauth2Check)
	r.Handle(protocol.MethodLobbyOauth2Signup, h.ok)
	r.Handle(protocol.MethodLobbyOpenidCheck, h.oauth2Check)
	r.Handle(protocol.MethodLobbyLoginSuccess, h.loginSuccess)
	r.Handle(protocol.MethodLobbyLoginBeat, h.ok)
	r.Handle(protocol.MethodLobbyLogout, h.ok)
	r.Handle(protocol.MethodLobbyCheckPrivacy, h.ok)
	r.Handle(protocol.MethodLobbyBindOauth2, h.ok)
	r.Handle(protocol.MethodLobbyFetchLastPrivacy, h.fetchLastPrivacy)
	r.Handle(protocol.MethodLobbyFetchInfo, h.fetchInfo)
	r.Handle(protocol.MethodLobbyFetchConnectionInfo, h.fetchConnectionInfo)

	registerEmptySurface(r, reg, log)
}

type handler struct {
	user *user.Service
	log  *slog.Logger
}

func (h *handler) requestConnection(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResRequestConnection, &resRequestConnection{
		Result:    1,
		Timestamp: uint32(now()),
	})
}

func (h *handler) heartbeat(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResHeartbeat, &resHeartbeat{})
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
	if err := ctx.Reg.DecodeInto(requestType(ctx.Method), ctx.Payload, &req); err != nil {
		return err
	}
	username := req.Account
	if username == "" {
		username = req.Email
	}
	if username == "" {
		username = "default"
	}

	acc, err := h.user.Login(context.Background(), username, req.Password)
	if err != nil {
		h.log.Error("login failed", "user", username, "err", err)
		return err
	}
	ctx.Session.SetAccountID(acc.ID)
	home, err := h.user.Home(context.Background(), acc.ID)
	if err != nil {
		return err
	}

	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResLogin, &resLogin{
		AccountID:             uint32(acc.ID),
		Account:               h.accountRPC(home),
		AccessToken:           accessToken(acc.ID),
		HasUnreadAnnouncement: false,
		Country:               "chs",
		IsIDCardAuthed:        true,
		SignupTime:            uint32(acc.CreatedAt),
	})
}

func (h *handler) oauth2Auth(ctx *router.Context) error {
	acc, err := h.user.Get(context.Background(), 1)
	if err != nil {
		return err
	}
	ctx.Session.SetAccountID(acc.ID)
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResOauth2Auth, &resOauth2Auth{
		AccessToken: accessToken(acc.ID),
	})
}

func (h *handler) oauth2Check(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResOauth2Check, &resOauth2Check{HasAccount: true})
}

// loginSuccess finalises login and pushes initial account state.
func (h *handler) loginSuccess(ctx *router.Context) error {
	if err := ctx.Session.Respond(ctx.MsgID, protocol.TypeResCommon, empty{}); err != nil {
		return err
	}
	return ctx.Session.Notify(protocol.NotifyAccountUpdate, &notifyAccountUpdate{
		Update: h.updatePayload(ctx.Session.AccountID()),
	})
}

func (h *handler) fetchInfo(ctx *router.Context) error {
	home, err := h.user.Home(context.Background(), ctx.Session.AccountID())
	if err != nil {
		return err
	}
	dto := &resFetchInfo{}
	dto.ServerTime.ServerTime = now()
	dto.CharacterInfo.Characters = h.charRPCs(home)
	if len(home.Characters) > 0 {
		dto.CharacterInfo.MainCharacterID = uint32(home.Characters[0].CharID)
		dto.CharacterInfo.Skins = characterSkins(home)
	}
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResFetchInfo, dto)
}

func (h *handler) fetchLastPrivacy(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResFetchLastPrivacy, &resFetchLastPrivacy{
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
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResConnectionInfo, info)
}

func (h *handler) ok(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResCommon, empty{})
}

// requestType maps a login-family method to its proto request type.
func requestType(method string) string {
	switch method {
	case protocol.MethodLobbyOauth2Login:
		return protocol.TypeReqOauth2Login
	case protocol.MethodLobbyEmailLogin:
		return protocol.TypeReqEmailLogin
	case protocol.MethodLobbyFastLogin:
		return protocol.TypeReqFastLogin
	default:
		return protocol.TypeReqLogin
	}
}
