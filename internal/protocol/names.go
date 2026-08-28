// Package protocol 常量中心：Majsoul 协议的方法名、消息类型名、notify 名与
// action 名的唯一来源。业务代码一律引用常量，禁止散落字符串字面量。
package protocol

// ── 命名空间前缀 ──────────────────────────────────────────────────────

const (
	TypePrefix            = "lq."
	LobbyServicePrefix    = ".lq.Lobby"
	FastTestServicePrefix = ".lq.FastTest"
	RouteServicePrefix    = ".lq.Route"
)

// ── 方法名（RPC 路由键）───────────────────────────────────────────────

const (
	MethodRouteRequestConnection = ".lq.Route.requestConnection"
	MethodRouteHeartbeat         = ".lq.Route.heartbeat"

	MethodLobbyPrepareLogin        = ".lq.Lobby.prepareLogin"
	MethodLobbySignup              = ".lq.Lobby.signup"
	MethodLobbyLogin               = ".lq.Lobby.login"
	MethodLobbyEmailLogin          = ".lq.Lobby.emailLogin"
	MethodLobbyFastLogin           = ".lq.Lobby.fastLogin"
	MethodLobbyOauth2Login         = ".lq.Lobby.oauth2Login"
	MethodLobbyOauth2Auth          = ".lq.Lobby.oauth2Auth"
	MethodLobbyOauth2Check         = ".lq.Lobby.oauth2Check"
	MethodLobbyOauth2Signup        = ".lq.Lobby.oauth2Signup"
	MethodLobbyOpenidCheck         = ".lq.Lobby.openidCheck"
	MethodLobbyLoginSuccess        = ".lq.Lobby.loginSuccess"
	MethodLobbyLoginBeat           = ".lq.Lobby.loginBeat"
	MethodLobbyLogout              = ".lq.Lobby.logout"
	MethodLobbyCheckPrivacy        = ".lq.Lobby.checkPrivacy"
	MethodLobbyBindOauth2          = ".lq.Lobby.bindOauth2"
	MethodLobbyFetchInfo           = ".lq.Lobby.fetchInfo"
	MethodLobbyFetchLastPrivacy    = ".lq.Lobby.fetchLastPrivacy"
	MethodLobbyFetchConnectionInfo = ".lq.Lobby.fetchConnectionInfo"

	MethodLobbyFetchAnnouncement    = ".lq.Lobby.fetchAnnouncement"
	MethodLobbyOpenAllRewardItem    = ".lq.Lobby.openAllRewardItem"
	MethodLobbyFetchQuestionnaire   = ".lq.Lobby.fetchQuestionnaireList"
	MethodLobbyFetchChallengeInfo   = ".lq.Lobby.fetchChallengeInfo"
	MethodLobbyFetchChallengeSeason = ".lq.Lobby.fetchChallengeSeason"
	MethodLobbyFetchSeerReportList  = ".lq.Lobby.fetchSeerReportList"
	MethodLobbyFetchReviveCoin      = ".lq.Lobby.fetchReviveCoinInfo"
	MethodLobbyFetchDailyTask       = ".lq.Lobby.fetchDailyTask"
	MethodLobbyFetchCommentSetting  = ".lq.Lobby.fetchCommentSetting"
	MethodLobbyFetchAchievementRate = ".lq.Lobby.fetchAchievementRate"
	MethodLobbyFetchAchievement     = ".lq.Lobby.fetchAchievement"
	MethodLobbyOpenGacha            = ".lq.Lobby.openGacha"
	MethodLobbyFetchRollingNotice   = ".lq.Lobby.fetchRollingNotice"
	MethodLobbyFetchActivity        = ".lq.Lobby.fetchActivity"

	MethodLobbyCreateRoom    = ".lq.Lobby.createRoom"
	MethodLobbyJoinRoom      = ".lq.Lobby.joinRoom"
	MethodLobbyLeaveRoom     = ".lq.Lobby.leaveRoom"
	MethodLobbyReadyPlay     = ".lq.Lobby.readyPlay"
	MethodLobbyStartRoom     = ".lq.Lobby.startRoom"
	MethodLobbyAddRoomRobot  = ".lq.Lobby.addRoomRobot"
	MethodLobbyKickPlayer    = ".lq.Lobby.roomKickPlayer"
	MethodLobbyFetchRoom     = ".lq.Lobby.fetchRoom"
	MethodLobbyFetchRoomList = ".lq.Lobby.fetchRoomList"

	MethodLobbyMatchGame          = ".lq.Lobby.matchGame"
	MethodLobbyCancelMatch        = ".lq.Lobby.cancelMatch"
	MethodLobbyMatchShiLian       = ".lq.Lobby.matchShiLian"
	MethodLobbyStartUnifiedMatch  = ".lq.Lobby.startUnifiedMatch"
	MethodLobbyCancelUnifiedMatch = ".lq.Lobby.cancelUnifiedMatch"
)

// FastTest 对局方法（对局会话层落地前由 surface 空响应兜底）。
const (
	MethodFastTestAuthGame          = ".lq.FastTest.authGame"
	MethodFastTestSyncGame          = ".lq.FastTest.syncGame"
	MethodFastTestInputOperation    = ".lq.FastTest.inputOperation"
	MethodFastTestInputChiPengGang  = ".lq.FastTest.inputChiPengGang"
	MethodFastTestConfirmNewRound   = ".lq.FastTest.confirmNewRound"
	MethodFastTestEnterGame         = ".lq.FastTest.enterGame"
	MethodFastTestCheckNetworkDelay = ".lq.FastTest.checkNetworkDelay"
	MethodFastTestFinishSyncGame    = ".lq.FastTest.finishSyncGame"
	MethodFastTestTerminateGame     = ".lq.FastTest.terminateGame"
)

// ── 消息类型名 ────────────────────────────────────────────────────────

const (
	TypeError = "lq.Error"

	TypeReqCommon = "lq.ReqCommon"
	TypeResCommon = "lq.ResCommon"

	TypeReqLogin       = "lq.ReqLogin"
	TypeReqEmailLogin  = "lq.ReqEmailLogin"
	TypeReqFastLogin   = "lq.ReqFastLogin"
	TypeReqOauth2Login = "lq.ReqOauth2Login"
	TypeResLogin       = "lq.ResLogin"
	TypeResOauth2Auth  = "lq.ResOauth2Auth"
	TypeResOauth2Check = "lq.ResOauth2Check"

	TypeResRequestConnection = "lq.ResRequestConnection"
	TypeResHeartbeat         = "lq.ResHeartbeat"
	TypeResFetchInfo         = "lq.ResFetchInfo"
	TypeResFetchLastPrivacy  = "lq.ResFetchLastPrivacy"
	TypeResConnectionInfo    = "lq.ResConnectionInfo"

	TypeReqSignupAccount = "lq.ReqSignupAccount"
	TypeResSignupAccount = "lq.ResSignupAccount"
	TypeResOauth2Signup  = "lq.ResOauth2Signup"
	TypeResFastLogin     = "lq.ResFastLogin"

	TypeResAnnouncement         = "lq.ResAnnouncement"
	TypeResOpenAllRewardItem    = "lq.ResOpenAllRewardItem"
	TypeResFetchQuestionnaire   = "lq.ResFetchQuestionnaireList"
	TypeResFetchChallengeInfo   = "lq.ResFetchChallengeInfo"
	TypeResChallengeSeasonInfo  = "lq.ResChallengeSeasonInfo"
	TypeResFetchSeerReportList  = "lq.ResFetchSeerReportList"
	TypeResReviveCoinInfo       = "lq.ResReviveCoinInfo"
	TypeResDailyTask            = "lq.ResDailyTask"
	TypeResCommentSetting       = "lq.ResCommentSetting"
	TypeResFetchAchievementRate = "lq.ResFetchAchievementRate"
	TypeResAchievement          = "lq.ResAchievement"
	TypeResOpenGacha            = "lq.ResOpenGacha"
	TypeResFetchRollingNotice   = "lq.ResFetchRollingNotice"
	TypeResActivityList         = "lq.ResActivityList"
	TypeResFetchChallengeTop    = "lq.ResFetchChallengeTop"

	TypeReqCreateRoom     = "lq.ReqCreateRoom"
	TypeResCreateRoom     = "lq.ResCreateRoom"
	TypeReqJoinRoom       = "lq.ReqJoinRoom"
	TypeResJoinRoom       = "lq.ResJoinRoom"
	TypeResSelfRoom       = "lq.ResSelfRoom"
	TypeReqRoomReady      = "lq.ReqRoomReady"
	TypeReqRoomStart      = "lq.ReqRoomStart"
	TypeReqAddRoomRobot   = "lq.ReqAddRoomRobot"
	TypeReqRoomKickPlayer = "lq.ReqRoomKickPlayer"
)

// ── Notify 名 ─────────────────────────────────────────────────────────

const (
	NotifyAccountUpdate  = ".lq.NotifyAccountUpdate"
	NotifyMatchGameStart = ".lq.NotifyMatchGameStart"
	NotifyRoomGameStart  = ".lq.NotifyRoomGameStart"
)

// ── Action（对局动作，Game 层使用）───────────────────────────────────

const (
	ActionPrototypeNamespace = ".lq.ActionPrototype"
	TypeActionPrototype      = "lq.ActionPrototype"
	ActionNewRound           = "ActionNewRound"
	ActionDealTile           = "ActionDealTile"
	ActionDiscardTile        = "ActionDiscardTile"
	ActionChiPengGang        = "ActionChiPengGang"
	ActionAnGangAddGang      = "ActionAnGangAddGang"
	ActionHule               = "ActionHule"
	ActionLiuJu              = "ActionLiuJu"
)
