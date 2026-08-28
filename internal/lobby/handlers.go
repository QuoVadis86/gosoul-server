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
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/qy-info/gosoul/internal/gacha"
	"github.com/qy-info/gosoul/internal/paipu"
	"github.com/qy-info/gosoul/internal/protocol"
	"github.com/qy-info/gosoul/internal/room"
	"github.com/qy-info/gosoul/internal/router"
	"github.com/qy-info/gosoul/internal/shop"
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
func Handlers(svc *user.Service, log *slog.Logger, r *router.Router, reg *protocol.Registry, rooms *room.Service, gameAddr string, pp *paipu.Service) {
	h := &handler{user: svc, log: log, rooms: rooms, gameAddr: gameAddr, paipu: pp}

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
	r.Handle(protocol.MethodLobbyFetchAchievement, h.fetchAchievement)
	r.Handle(protocol.MethodLobbyOpenGacha, h.openGacha)
	r.Handle(protocol.MethodLobbyBuyFromShop, h.buyFromShop)
	r.Handle(protocol.MethodLobbyFetchRollingNotice, h.fetchRollingNotice)
	r.Handle(protocol.MethodLobbyFetchActivity, h.fetchActivity)
	r.Handle(protocol.MethodLobbyFetchMailInfo, h.fetchMailInfo)
	r.Handle(protocol.MethodLobbyFetchMaintainNotice, h.fetchMaintainNotice)
	r.Handle(protocol.MethodLobbyFetchIDCardInfo, h.fetchIDCardInfo)
	r.Handle(protocol.MethodLobbyFetchGameRecord, h.fetchGameRecord)

	r.Handle(protocol.MethodLobbyCreateRoom, h.createRoom)
	r.Handle(protocol.MethodLobbyJoinRoom, h.joinRoom)
	r.Handle(protocol.MethodLobbyLeaveRoom, h.leaveRoom)
	r.Handle(protocol.MethodLobbyReadyPlay, h.readyPlay)
	r.Handle(protocol.MethodLobbyStartRoom, h.startRoom)
	r.Handle(protocol.MethodLobbyAddRoomRobot, h.addRoomRobot)
	r.Handle(protocol.MethodLobbyKickPlayer, h.kickPlayer)
	r.Handle(protocol.MethodLobbyFetchRoom, h.fetchRoom)
	r.Handle(protocol.MethodLobbyMatchGame, h.matchGame)
	r.Handle(protocol.MethodLobbyCancelMatch, h.cancelMatch)

	registerEmptySurface(r, reg, log)
}

type handler struct {
	user     *user.Service
	log      *slog.Logger
	rooms    *room.Service
	gameAddr string
	gacha    *gacha.Service
	shop     *shop.Service
	paipu    *paipu.Service
}

// ensureShop lazily builds the currency shop over the user service.
func (h *handler) ensureShop() *shop.Service {
	if h.shop == nil {
		h.shop = shop.New(userShop{h.user}, shopCatalog())
	}
	return h.shop
}

func shopCatalog() []shop.Goods {
	return []shop.Goods{
		{ID: 1, Cost: 100, Currency: shop.Diamond, RewardID: 100, RewardCount: 5000}, // 5000 gold
		{ID: 2, Cost: 10, Currency: shop.SkinTicket, RewardID: 100, RewardCount: 500},
	}
}

// userShop adapts the user domain to the shop teller contract.
type userShop struct{ svc *user.Service }

func (u userShop) Balance(ctx context.Context, id int64) (int64, int64, int64, error) {
	w, err := u.svc.Balance(ctx, id)
	if err != nil {
		return 0, 0, 0, err
	}
	return w.Gold, w.Diamond, w.SkinTicket, nil
}
func (u userShop) Charge(ctx context.Context, id, g, d, t int64) error {
	return u.svc.Grant(ctx, id, g, d, t)
}
func (u userShop) Grant(ctx context.Context, id, g, d, t int64) error {
	return u.svc.Grant(ctx, id, g, d, t)
}

// ensureGacha lazily builds the character lottery over the user service.
func (h *handler) ensureGacha() *gacha.Service {
	if h.gacha == nil {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		h.gacha = gacha.New(userGacha{h.user}, gacha.Pool{
			Characters: []int64{200001, 200002, 200003, 200101, 200102},
		}, rng)
	}
	return h.gacha
}

// userGacha adapts the user domain to the gacha drawer contract.
type userGacha struct{ svc *user.Service }

func (g userGacha) Balance(ctx context.Context, id int64) (int64, error) {
	w, err := g.svc.Balance(ctx, id)
	if err != nil {
		return 0, err
	}
	return w.Diamond, nil
}
func (g userGacha) Charge(ctx context.Context, id, delta int64) error {
	return g.svc.Grant(ctx, id, 0, delta, 0)
}
func (g userGacha) GrantCharacter(ctx context.Context, id, charID int64) error {
	return g.svc.GrantCharacter(ctx, id, charID)
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
	return nil
}

func (h *handler) oauth2Auth(ctx *router.Context) error {
	// 真实第三方授权未接入；本地场景不签发 oauth token。
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResOauth2Auth, &resOauth2Auth{
		Error: errorCode(2000),
	})
}

// oauth2Signup 在 oauth2Check(false) 后由客户端调用：以客户端本地持久 token
// 作为账号键注册（幂等），使游客身份跨会话稳定。
func (h *handler) oauth2Signup(ctx *router.Context) error {
	var req struct {
		Type        uint32 `json:"type"`
		Email       string `json:"email"`
		AccessToken string `json:"accessToken"`
	}
	if err := ctx.Reg.DecodeInto("lq.ReqOauth2Signup", ctx.Payload, &req); err != nil {
		return err
	}
	key := req.AccessToken
	if key == "" {
		key = req.Email
	}
	if key == "" {
		key = fmt.Sprintf("oauth_%d", time.Now().Unix())
	}
	// 复用 token 建号路径：tokenKey(key) 作 username，同 key 幂等。
	acc, err := h.user.SignupByToken(context.Background(), tokenKey(key))
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

// fetchAchievement reports recorded achievement progress for the account.
func (h *handler) fetchAchievement(ctx *router.Context) error {
	achvs, err := h.user.Achievements(context.Background(), ctx.Session.AccountID())
	if err != nil {
		return err
	}
	res := &resAchievement{Error: &errBody{}}
	for _, a := range achvs {
		res.Progresses = append(res.Progresses, achievementProgress{
			ID:       uint32(a.AchieveID),
			Counter:  uint32(a.Progress),
			Achieved: a.Progress > 0,
			Rewarded: a.Rewarded > 0,
		})
	}
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResAchievement, res)
}

// openGacha performs a paid character lottery draw.
func (h *handler) openGacha(ctx *router.Context) error {
	var req struct {
		Count uint32 `json:"count"`
	}
	if err := ctx.Reg.DecodeInto("lq.ReqOpenGacha", ctx.Payload, &req); err != nil {
		return err
	}
	res, err := h.ensureGacha().Open(context.Background(), ctx.Session.AccountID(), req.Count)
	if errors.Is(err, gacha.ErrNotEnough) {
		return ctx.Session.Respond(ctx.MsgID, protocol.TypeResOpenGacha, &resOpenGacha{Error: errorCode(4001)})
	}
	if err != nil {
		return err
	}
	out := &resOpenGacha{Error: &errBody{}}
	for _, id := range res.ResultList {
		out.ResultList = append(out.ResultList, uint32(id))
	}
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResOpenGacha, out)
}

// buyFromShop purchases a catalog item with the account's currency.
func (h *handler) buyFromShop(ctx *router.Context) error {
	var req struct {
		GoodsID uint32 `json:"goodsId"`
		Count   uint32 `json:"count"`
	}
	if err := ctx.Reg.DecodeInto("lq.ReqBuyFromShop", ctx.Payload, &req); err != nil {
		return err
	}
	slots, err := h.ensureShop().Purchase(context.Background(), ctx.Session.AccountID(), req.GoodsID, int64(req.Count))
	if errors.Is(err, shop.ErrNoGoods) || errors.Is(err, shop.ErrNotEnough) {
		return ctx.Session.Respond(ctx.MsgID, protocol.TypeResBuyFromShop, &resBuyFromShop{Error: errorCode(4001)})
	}
	if err != nil {
		return err
	}
	out := &resBuyFromShop{Error: &errBody{}}
	for _, s := range slots {
		out.Rewards = append(out.Rewards, rewardSlot{ID: s.ID, Count: s.Count})
	}
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResBuyFromShop, out)
}

func (h *handler) fetchRollingNotice(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResFetchRollingNotice, &resFetchRollingNotice{
		Error: &errBody{},
		Notice: &rollingNotice{
			Content: "欢迎来到 gosoul 私服！日麻环境已就绪。",
		},
	})
}

func (h *handler) fetchActivity(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResActivityList, &resActivityList{
		Error: &errBody{},
		ActivityList: []activity{
			{ActivityID: 1, Type: "新手"},
		},
	})
}

// fetchMailInfo returns a static welcome mail.
func (h *handler) fetchMailInfo(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResMailInfo, &resMailInfo{
		Error: &errBody{},
		Mails: []mail{{
			MailID:     1,
			State:      0,
			Title:      "欢迎",
			Content:    "感谢使用 gosoul 私服。祝你麻将愉快。",
			CreateTime: uint32(now()),
			ExpireTime: uint32(now() + 86400*30),
		}},
	})
}

// fetchMaintainNotice reports no active maintenance.
func (h *handler) fetchMaintainNotice(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResFetchMaintainNotice, &resFetchMaintainNotice{Error: &errBody{}})
}

// fetchIDCardInfo reports the account's real-name state (unverified per default).
func (h *handler) fetchIDCardInfo(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResIDCardInfo, &resIDCardInfo{Error: &errBody{}, IsAuthed: false, Country: "chs"})
}

// fetchGameRecord returns a stored paipu replay summary for a game uuid.
func (h *handler) fetchGameRecord(ctx *router.Context) error {
	var req struct {
		GameUUID string `json:"gameUuid"`
	}
	if err := ctx.Reg.DecodeInto("lq.ReqGameRecord", ctx.Payload, &req); err != nil {
		return err
	}
	if h.paipu == nil {
		return ctx.Session.Respond(ctx.MsgID, protocol.TypeResGameRecord, &resGameRecord{Error: errorCode(4001)})
	}
	rec, err := h.paipu.Get(context.Background(), req.GameUUID)
	if err != nil {
		return ctx.Session.Respond(ctx.MsgID, protocol.TypeResGameRecord, &resGameRecord{Error: errorCode(2003)})
	}
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResGameRecord, &resGameRecord{
		Error: &errBody{},
		Head: &recordGame{
			UUID:      rec.UUID,
			StartTime: uint32(rec.CreatedAt.Unix()),
		},
		DataURL: "local",
	})
}

type notifyMatchGameStart struct {
	GameURL      string `json:"gameUrl"`
	ConnectToken string `json:"connectToken"`
	GameUUID     string `json:"gameUuid"`
	MatchModeID  uint32 `json:"matchModeId"`
	Location     string `json:"location"`
}

// matchGame answers a ranked-casual queue request by acknowledging immediately
// and pushing NotifyMatchGameStart shortly after, mirroring the reference.
func (h *handler) matchGame(ctx *router.Context) error {
	var req struct {
		MatchType uint32 `json:"matchType"`
		Mode      uint32 `json:"mode"`
	}
	_ = ctx.Reg.DecodeInto("lq.ReqJoinMatchQueue", ctx.Payload, &req)
	accountID := uint32(ctx.Session.AccountID())
	if accountID == 0 {
		accountID = 10001
	}
	h.log.Info("match requested", "account", accountID, "type", req.MatchType, "mode", req.Mode)
	if err := ctx.Session.Respond(ctx.MsgID, protocol.TypeResCommon, empty{}); err != nil {
		return err
	}
	token := fmt.Sprintf("local-connect-token-%d", accountID)
	uuid := fmt.Sprintf("local-game-uuid-%d-%d", accountID, now())
	go func() {
		time.Sleep(1200 * time.Millisecond)
		_ = ctx.Session.Notify(protocol.NotifyMatchGameStart, &notifyMatchGameStart{
			GameURL:      h.gameAddr,
			ConnectToken: token,
			GameUUID:     uuid,
			MatchModeID:  1,
			Location:     "local",
		})
	}()
	return nil
}

func (h *handler) cancelMatch(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResCommon, empty{})
}

func (h *handler) createRoom(ctx *router.Context) error {
	var req struct {
		PlayerCount uint32         `json:"playerCount"`
		Mode        map[string]any `json:"mode"`
		PublicLive  bool           `json:"publicLive"`
	}
	if err := ctx.Reg.DecodeInto(protocol.TypeReqCreateRoom, ctx.Payload, &req); err != nil {
		return err
	}
	if req.PlayerCount == 0 {
		req.PlayerCount = 4
	}
	accountID := uint32(ctx.Session.AccountID())
	r := h.rooms.Create(accountID, req.PlayerCount, room.Mode{Mode: modeInt(req.Mode), DetailRule: detailRules(req.Mode)}, req.PublicLive)
	res := &resCreateRoom{Error: &errBody{}, Room: h.roomView(r)}
	if b, err := json.Marshal(res); err == nil {
		h.log.Info("createRoom payload", "json", string(b))
	}
	h.log.Info("createRoom decision", "account", accountID, "room", r.ID, "players", len(r.Players))
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResCreateRoom, res)
}

func detailRules(m map[string]any) map[string]any {
	if v, ok := m["detailRule"].(map[string]any); ok {
		return v
	}
	if v, ok := m["detail_rule"].(map[string]any); ok {
		return v
	}
	return nil
}

func (h *handler) joinRoom(ctx *router.Context) error {
	var req struct {
		RoomID uint32 `json:"roomId"`
	}
	if err := ctx.Reg.DecodeInto(protocol.TypeReqJoinRoom, ctx.Payload, &req); err != nil {
		return err
	}
	r, err := h.rooms.Join(req.RoomID, uint32(ctx.Session.AccountID()))
	if err != nil {
		return ctx.Session.Respond(ctx.MsgID, protocol.TypeResJoinRoom, &resJoinRoom{Error: errorCode(1001)})
	}
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResJoinRoom, &resJoinRoom{Error: &errBody{}, Room: h.roomView(r)})
}

func (h *handler) leaveRoom(ctx *router.Context) error {
	var req struct {
		RoomID uint32 `json:"roomId"`
	}
	if err := ctx.Reg.DecodeInto("lq.ReqLeaveRoom", ctx.Payload, &req); err != nil {
		return h.ok(ctx)
	}
	_ = h.rooms.Leave(req.RoomID, uint32(ctx.Session.AccountID()))
	return h.ok(ctx)
}

func (h *handler) readyPlay(ctx *router.Context) error {
	var req struct {
		Ready bool `json:"ready"`
	}
	if err := ctx.Reg.DecodeInto(protocol.TypeReqRoomReady, ctx.Payload, &req); err != nil {
		return err
	}
	if _, err := h.rooms.SetReady(h.rooms.RoomOf(uint32(ctx.Session.AccountID())), uint32(ctx.Session.AccountID()), req.Ready); err != nil {
		return h.ok(ctx)
	}
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResCommon, empty{})
}

func (h *handler) startRoom(ctx *router.Context) error {
	var req struct {
		RoomID uint32 `json:"roomId"`
	}
	if err := ctx.Reg.DecodeInto(protocol.TypeReqRoomStart, ctx.Payload, &req); err != nil {
		return h.ok(ctx)
	}
	if _, err := h.rooms.Start(req.RoomID); err != nil {
		return h.ok(ctx)
	}
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResCommon, empty{})
}

func (h *handler) addRoomRobot(ctx *router.Context) error {
	var req struct {
		RoomID uint32 `json:"roomId"`
	}
	if err := ctx.Reg.DecodeInto(protocol.TypeReqAddRoomRobot, ctx.Payload, &req); err != nil {
		return h.ok(ctx)
	}
	roomID := req.RoomID
	if roomID == 0 {
		roomID = h.rooms.RoomOf(uint32(ctx.Session.AccountID()))
	}
	if _, err := h.rooms.AddRobot(roomID); err != nil {
		return h.ok(ctx)
	}
	return h.ok(ctx)
}

func (h *handler) kickPlayer(ctx *router.Context) error {
	return h.ok(ctx)
}

func (h *handler) fetchRoom(ctx *router.Context) error {
	roomID := h.rooms.RoomOf(uint32(ctx.Session.AccountID()))
	r, err := h.rooms.Get(roomID)
	if err != nil {
		return ctx.Session.Respond(ctx.MsgID, protocol.TypeResSelfRoom, &resSelfRoom{Error: errorCode(1001)})
	}
	return ctx.Session.Respond(ctx.MsgID, protocol.TypeResSelfRoom, &resSelfRoom{Error: &errBody{}, Room: h.roomView(r)})
}

func modeInt(m map[string]any) uint32 {
	if v, ok := m["mode"].(float64); ok {
		return uint32(v)
	}
	return 1
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
