// Package lobby implements the account-facing lobby RPC handlers: route
// connection, login/registration and home data. Handlers translate liqi
// payloads (DTOs) onto the lobby domain service.
package lobby

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/qy-info/gosoul/internal/protocol"
	"github.com/qy-info/gosoul/internal/router"
	"github.com/qy-info/gosoul/internal/user"
)

func tokenKey(token string) string { return "t:" + token }

// usernameFromToken extracts the account id from a local-token-N.
func usernameFromToken(token string) int64 {
	if len(token) < 12 || token[:12] != "local-token-" {
		return 0
	}
	id, err := strconv.ParseInt(token[12:], 10, 64)
	if err != nil {
		return 0
	}
	return id
}

// Handlers registers every lobby RPC the client uses at login.
func Handlers(svc *user.Service, log *slog.Logger, r *router.Router, reg *protocol.Registry) {
	h := &handler{user: svc, log: log}

	r.Handle(protocol.MethodRouteRequestConnection, h.requestConnection)
	r.Handle(protocol.MethodRouteHeartbeat, h.heartbeat)
	r.Handle(protocol.MethodLobbyPrepareLogin, h.ok)
	r.Handle(protocol.MethodLobbySignup, h.signup)
	r.Handle(protocol.MethodLobbyLogin, h.login)
	r.Handle(protocol.MethodLobbyEmailLogin, h.login)
	r.Handle(protocol.MethodLobbyFastLogin, h.login)
	r.Handle(protocol.MethodLobbyOauth2Login, h.login)
	r.Handle(protocol.MethodLobbyOauth2Auth, h.oauth2Auth)
	r.Handle(protocol.MethodLobbyOauth2Check, h.oauth2Check)
	r.Handle(protocol.MethodLobbyOauth2Signup, h.oauth2Signup)
	r.Handle(protocol.MethodLobbyOpenidCheck, h.oauth2Check)
	r.Handle(protocol.MethodLobbyLoginSuccess, h.loginSuccess)
	r.Handle(protocol.MethodLobbyLoginBeat, h.ok)
	r.Handle(protocol.MethodLobbyLogout, h.ok)
	r.Handle(protocol.MethodLobbyCheckPrivacy, h.ok)
	r.Handle(protocol.MethodLobbyBindOauth2, h.ok)
	r.Handle(protocol.MethodLobbyFetchLastPrivacy, h.fetchLastPrivacy)
	r.Handle(protocol.MethodLobbyFetchInfo, h.fetchInfo)
	r.Handle(protocol.MethodLobbyFetchConnectionInfo, h.fetchConnectionInfo)
	r.Handle(protocol.MethodLobbyFetchAnnouncement, h.fetchAnnouncement)
	r.Handle(protocol.MethodLobbyOpenAllRewardItem, h.openAllRewardItem)
	r.Handle(protocol.MethodLobbyFetchQuestionnaire, h.fetchQuestionnaire)
	r.Handle(protocol.MethodLobbyFetchChallengeInfo, h.fetchChallengeInfo)
	r.Handle(protocol.MethodLobbyFetchChallengeSeason, h.fetchChallengeSeason)
	r.Handle(protocol.MethodLobbyFetchSeerReportList, h.fetchSeerReportList)
	r.Handle(protocol.MethodLobbyFetchReviveCoin, h.fetchReviveCoin)
	r.Handle(protocol.MethodLobbyFetchDailyTask, h.fetchDailyTask)
	r.Handle(protocol.MethodLobbyFetchCommentSetting, h.fetchCommentSetting)
	r.Handle(protocol.MethodLobbyFetchAchievementRate, h.fetchAchievementRate)
	r.Handle(protocol.MethodLobbyFetchRollingNotice, h.fetchRollingNotice)
	r.Handle(protocol.MethodLobbyFetchActivity, h.fetchActivity)

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

// signup 注册新账号；重名返回错误。
func (h *handler) signup(ctx *router.Context) error {
	var req struct {
		Account  string `json:"account"`
		Password string `json:"password"`
	}
	if err := ctx.Reg.DecodeInto(protocol.TypeReqSignupAccount, ctx.Payload, &req); err != nil {
		return err
	}
	if req.Account == "" {
		return ctx.Session.Respond(ctx.MsgID, protocol.TypeResSignupAccount, &resSignupAccount{Error: errorCode(2001)})
	}
	if _, err := h.user.Signup(context.Background(), req.Account, req.Password, ""); errors.Is(err, user.ErrTaken) {
		return ctx.Session.Respond(ctx.MsgID, protocol.TypeResSignupAccount, &resSignupAccount{Error: errorCode(2001)})
	} else if err != nil {
		return err
	}
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResSignupAccount, &resSignupAccount{Error: &errBody{}})
}

type reqLogin struct {
	Account     string `json:"account"`
	Password    string `json:"password"`
	Email       string `json:"email"`
	AccessToken string `json:"accessToken"`
}

// login 校验账号与密码；不存在或密码错都是真实错误（注册走 signup）。
func (h *handler) login(ctx *router.Context) error {
	var req reqLogin
	if err := ctx.Reg.DecodeInto(requestType(ctx.Method), ctx.Payload, &req); err != nil {
		h.log.Error("login decode failed", "method", ctx.Method, "err", err)
		return err
	}
	h.log.Info("login decision", "method", ctx.Method, "token", req.AccessToken, "account", req.Account, "email", req.Email)
	// token-based 登录（oauth2Login/fastLogin）：客户端游客 token / 旧 local-token-N
	if token := req.AccessToken; token != "" {
		acc, err := h.user.LoginByToken(context.Background(), tokenKey(token))
		if errors.Is(err, user.ErrNotFound) && strings.HasPrefix(token, "local-token-") {
			acc, err = h.user.Get(context.Background(), usernameFromToken(token))
		}
		if err != nil {
			return ctx.Session.Respond(ctx.MsgID, protocol.TypeResLogin, &resLogin{Error: errorCode(2001)})
		}
		ctx.Session.SetAccountID(acc.ID)
		home, err := h.user.Home(context.Background(), acc.ID)
		if err != nil {
			return err
		}
		return h.respondLogin(ctx, acc, home)
	}

	username := req.Account
	if username == "" {
		username = req.Email
	}
	if username == "" {
		return ctx.Session.Respond(ctx.MsgID, protocol.TypeResLogin, &resLogin{Error: errorCode(2001)})
	}

	acc, err := h.user.Login(context.Background(), username, req.Password)
	if errors.Is(err, user.ErrNotFound) {
		return ctx.Session.Respond(ctx.MsgID, protocol.TypeResLogin, &resLogin{Error: errorCode(2001)})
	}
	if errors.Is(err, user.ErrBadPassword) {
		return ctx.Session.Respond(ctx.MsgID, protocol.TypeResLogin, &resLogin{Error: errorCode(2002)})
	}
	if err != nil {
		h.log.Error("login failed", "user", username, "err", err)
		return err
	}
	ctx.Session.SetAccountID(acc.ID)
	home, err := h.user.Home(context.Background(), acc.ID)
	if err != nil {
		return err
	}
	return h.respondLogin(ctx, acc, home)
}

func (h *handler) respondLogin(ctx *router.Context, acc *user.Account, home *user.Home) error {
	res := &resLogin{
		Error:                 &errBody{},
		AccountID:             uint32(acc.ID),
		Account:               h.accountRPC(home),
		AccessToken:           accessToken(acc.ID),
		HasUnreadAnnouncement: false,
		Country:               "chs",
		IsIDCardAuthed:        true,
		SignupTime:            uint32(acc.CreatedAt),
	}
	if b, err := json.Marshal(res); err == nil {
		h.log.Info("reslogin payload", "json", string(b))
	}
	if err := ctx.Session.Respond(ctx.MsgID, protocol.TypeResLogin, res); err != nil {
		return err
	}
	// 登录成功后推送账号数值，客户端依赖此刷新大厅钱包/资源。
	return ctx.Session.Notify(protocol.NotifyAccountUpdate, &notifyAccountUpdate{
		Update: h.updatePayload(acc.ID),
	})
}

func (h *handler) oauth2Auth(ctx *router.Context) error {
	// 真实第三方授权未接入；本地场景不签发 oauth token。
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResOauth2Auth, &resOauth2Auth{
		Error: errorCode(2000),
	})
}

// oauth2Signup 在 oauth2Check(false) 后由客户端调用：注册并签发 token。
func (h *handler) oauth2Signup(ctx *router.Context) error {
	var req struct {
		Type        uint32 `json:"type"`
		Email       string `json:"email"`
		AccessToken string `json:"accessToken"`
	}
	if err := ctx.Reg.DecodeInto("lq.ReqOauth2Signup", ctx.Payload, &req); err != nil {
		return err
	}
	name := req.Email
	if name == "" {
		name = fmt.Sprintf("oauth_%d", time.Now().Unix())
	}
	acc, err := h.user.Signup(context.Background(), name, "", "")
	if errors.Is(err, user.ErrTaken) {
		return ctx.Session.Respond(ctx.MsgID, protocol.TypeResOauth2Signup, &resOauth2Signup{Error: errorCode(2001)})
	}
	if err != nil {
		return err
	}
	ctx.Session.SetAccountID(acc.ID)
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResOauth2Signup, &resOauth2Signup{
		AccessToken: accessToken(acc.ID),
	})
}

func (h *handler) oauth2Check(ctx *router.Context) error {
	var req struct {
		Type        uint32 `json:"type"`
		AccessToken string `json:"accessToken"`
	}
	if err := ctx.Reg.DecodeInto("lq.ReqOauth2Check", ctx.Payload, &req); err != nil {
		return err
	}
	hasAccount := false
	if req.AccessToken != "" {
		_, err := h.user.GetByUsernameToken(context.Background(), tokenKey(req.AccessToken))
		hasAccount = err == nil
	}
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResOauth2Check, &resOauth2Check{HasAccount: hasAccount})
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
	dto.CharacterInfo.Skins = characterSkins(home)
	dto.CharacterInfo.CharacterSort = h.charIDs(home)
	dto.CharacterInfo.MainCharacterID = 200001
	if len(home.Characters) > 0 {
		dto.CharacterInfo.MainCharacterID = uint32(home.Characters[0].CharID)
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

func (h *handler) fetchAnnouncement(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResAnnouncement, &resAnnouncement{Error: &errBody{}})
}

func (h *handler) openAllRewardItem(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResOpenAllRewardItem, &resOpenAllRewardItem{Error: &errBody{}})
}

func (h *handler) fetchQuestionnaire(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResFetchQuestionnaire, &resFetchQuestionnaireList{Error: &errBody{}})
}

func (h *handler) fetchChallengeInfo(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResFetchChallengeInfo, &resFetchChallengeInfo{Error: &errBody{}})
}

func (h *handler) fetchChallengeSeason(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResChallengeSeasonInfo, &resChallengeSeasonInfo{Error: &errBody{}})
}

func (h *handler) fetchSeerReportList(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResFetchSeerReportList, &resFetchSeerReportList{Error: &errBody{}})
}

func (h *handler) fetchReviveCoin(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResReviveCoinInfo, &resReviveCoinInfo{Error: &errBody{}})
}

func (h *handler) fetchDailyTask(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResDailyTask, &resDailyTask{Error: &errBody{}})
}

func (h *handler) fetchCommentSetting(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResCommentSetting, &resCommentSetting{Error: &errBody{}})
}

func (h *handler) fetchAchievementRate(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResFetchAchievementRate, &resFetchAchievementRate{Error: &errBody{}})
}

func (h *handler) fetchRollingNotice(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResFetchRollingNotice, &resFetchRollingNotice{Error: &errBody{}})
}

func (h *handler) fetchActivity(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResActivityList, &resActivityList{Error: &errBody{}})
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
