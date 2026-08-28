package lobby

import (
	"context"
	"fmt"
	"time"

	"github.com/qy-info/gosoul/internal/room"
	"github.com/qy-info/gosoul/internal/user"
)

func now() int64                         { return time.Now().Unix() }
func accessToken(accountID int64) string { return fmt.Sprintf("local-token-%d", accountID) }

// empty is the generic success envelope.
type empty struct{}

// resAnnouncement answers .lq.Lobby.fetchAnnouncement.
type resAnnouncement struct {
	Error         *errBody `json:"error"`
	Announcements []any    `json:"announcements"`
}

// resOpenAllRewardItem answers .lq.Lobby.openAllRewardItem.
type resOpenAllRewardItem struct {
	Error   *errBody `json:"error"`
	Results []any    `json:"results"`
}

// resFetchQuestionnaireList answers .lq.Lobby.fetchQuestionnaireList.
type resFetchQuestionnaireList struct {
	Error        *errBody `json:"error"`
	List         []any    `json:"list"`
	FinishedList []any    `json:"finishedList"`
}

// resFetchChallengeInfo answers .lq.Lobby.fetchChallengeInfo.
type resFetchChallengeInfo struct {
	Error *errBody `json:"error"`
}

// resChallengeSeasonInfo answers .lq.Lobby.fetchChallengeSeason.
type resChallengeSeasonInfo struct {
	Error *errBody `json:"error"`
}

// resFetchSeerReportList answers .lq.Lobby.fetchSeerReportList.
type resFetchSeerReportList struct {
	Error          *errBody `json:"error"`
	SeerReportList []any    `json:"seerReportList"`
}

// resReviveCoinInfo answers .lq.Lobby.fetchReviveCoinInfo.
type resReviveCoinInfo struct {
	Error     *errBody `json:"error"`
	HasGained bool     `json:"hasGained"`
}

// resDailyTask answers .lq.Lobby.fetchDailyTask.
type resDailyTask struct {
	Error             *errBody `json:"error"`
	Progresses        []any    `json:"progresses"`
	HasRefreshCount   bool     `json:"hasRefreshCount"`
	MaxDailyTaskCount uint32   `json:"maxDailyTaskCount"`
	RefreshCount      uint32   `json:"refreshCount"`
}

// resCommentSetting answers .lq.Lobby.fetchCommentSetting.
type resCommentSetting struct {
	Error          *errBody `json:"error"`
	CommentSetting struct {
		CommentAllowType uint32 `json:"commentAllowType"`
	} `json:"commentSetting"`
}

// resFetchAchievementRate answers .lq.Lobby.fetchAchievementRate.
type resFetchAchievementRate struct {
	Error *errBody `json:"error"`
}

// achievementProgress mirrors lq.AchievementProgress.
type achievementProgress struct {
	ID       uint32 `json:"id"`
	Counter  uint32 `json:"counter"`
	Achieved bool   `json:"achieved"`
	Rewarded bool   `json:"rewarded"`
}

// resAchievement answers .lq.Lobby.fetchAchievement.
type resAchievement struct {
	Error         *errBody              `json:"error"`
	Progresses    []achievementProgress `json:"progresses"`
	RewardedGroup []uint32              `json:"rewardedGroup"`
}

// resFetchRollingNotice answers .lq.Lobby.fetchRollingNotice.
type resFetchRollingNotice struct {
	Error *errBody `json:"error"`
}

// resActivityList answers .lq.Lobby.fetchActivity.
type resActivityList struct {
	Error        *errBody `json:"error"`
	ActivityList []any    `json:"activityList"`
}

// playerGameView is a room seat's view.
type playerGameView struct {
	AccountID uint32 `json:"accountId"`
	AvatarID  uint32 `json:"avatarId"`
	Nickname  string `json:"nickname"`
	Level     struct {
		ID uint32 `json:"id"`
	} `json:"level"`
	Level3 struct {
		ID uint32 `json:"id"`
	} `json:"level3"`
	Views    []any  `json:"views"`
	TeamName string `json:"teamName"`
}

// roomView is the Room projection the client renders for a friend room.
type roomView struct {
	RoomID         uint32           `json:"roomId"`
	OwnerID        uint32           `json:"ownerId"`
	Mode           map[string]any   `json:"mode"`
	MaxPlayerCount uint32           `json:"maxPlayerCount"`
	Persons        []playerGameView `json:"persons"`
	ReadyList      []uint32         `json:"readyList"`
	IsPlaying      bool             `json:"isPlaying"`
	PublicLive     bool             `json:"publicLive"`
	RobotCount     uint32           `json:"robotCount"`
	Robots         []playerGameView `json:"robots"`
	Positions      []uint32         `json:"positions"`
}

// resCreateRoom answers .lq.Lobby.createRoom.
type resCreateRoom struct {
	Error *errBody  `json:"error"`
	Room  *roomView `json:"room"`
}

// resJoinRoom answers .lq.Lobby.joinRoom.
type resJoinRoom struct {
	Error *errBody  `json:"error"`
	Room  *roomView `json:"room"`
}

// resSelfRoom answers .lq.Lobby.fetchRoom.
type resSelfRoom struct {
	Error *errBody  `json:"error"`
	Room  *roomView `json:"room"`
}

// errorCode is the shared way to answer with a non-zero error code.
func errorCode(code uint32) *errBody {
	return &errBody{Code: code}
}

// resSignupAccount answers .lq.Lobby.signup.
type resSignupAccount struct {
	Error *errBody `json:"error"`
}

// resOauth2Signup answers .lq.Lobby.oauth2Signup with the new token.
type resOauth2Signup struct {
	Error       *errBody `json:"error"`
	AccessToken string   `json:"accessToken"`
}

// ResRequestConnection: result/timestamp.
type resRequestConnection struct {
	Error     *errBody `json:"error"`
	Timestamp uint32   `json:"timestamp"`
	Result    uint32   `json:"result"`
}

type resHeartbeat struct {
	Error *errBody `json:"error"`
}

type errBody struct {
	Code uint32 `json:"code"`
}

type resOauth2Auth struct {
	Error       *errBody `json:"error"`
	AccessToken string   `json:"accessToken"`
}

type resOauth2Check struct {
	Error      *errBody `json:"error"`
	HasAccount bool     `json:"hasAccount"`
}

type privacyVersion struct {
	Type    int    `json:"type"`
	Version string `json:"version"`
}

type resFetchLastPrivacy struct {
	Error   *errBody         `json:"error"`
	Privacy []privacyVersion `json:"privacy"`
}

type resConnectionInfo struct {
	Error          *errBody `json:"error"`
	ClientEndpoint struct {
		Family  string `json:"family"`
		Address string `json:"address"`
		Port    uint32 `json:"port"`
	} `json:"clientEndpoint"`
}

// levelRPC is the level projection (id + score) used inside Account.
type levelRPC struct {
	ID    uint32 `json:"id"`
	Score uint32 `json:"score"`
}

// accountRPC is the nested Account projection the client renders.
type accountRPC struct {
	AccountID     uint32    `json:"accountId"`
	Nickname      string    `json:"nickname"`
	AvatarID      uint32    `json:"avatarId"`
	Level         *levelRPC `json:"level"`
	Level3        *levelRPC `json:"level3"`
	VIP           uint32    `json:"vip"`
	Title         uint32    `json:"title"`
	LoginTime     uint32    `json:"loginTime"`
	LogoutTime    uint32    `json:"logoutTime"`
	RoomID        uint32    `json:"roomId"`
	AntiAddiction struct {
		OnlineDuration uint32 `json:"onlineDuration"`
	} `json:"antiAddiction"`
	Email       string `json:"email"`
	PhoneVerify uint32 `json:"phoneVerify"`
	EmailVerify uint32 `json:"emailVerify"`
	AvatarFrame uint32 `json:"avatarFrame"`
	Gold        uint32 `json:"gold"`
	Diamond     uint32 `json:"diamond"`
	SkinTicket  uint32 `json:"skinTicket"`
	Signature   string `json:"signature"`
	Verified    uint32 `json:"verified"`
}

// resLogin answers .lq.Lobby.login / oauth2Login / fastLogin / emailLogin.
type resLogin struct {
	Error                 *errBody    `json:"error"`
	AccountID             uint32      `json:"accountId"`
	Account               *accountRPC `json:"account"`
	AccessToken           string      `json:"accessToken"`
	HasUnreadAnnouncement bool        `json:"hasUnreadAnnouncement"`
	Country               string      `json:"country"`
	IsIDCardAuthed        bool        `json:"isIdCardAuthed"`
	SignupTime            uint32      `json:"signupTime"`
}

// resFetchInfo answers .lq.Lobby.fetchInfo — the client's home payload.
type resFetchInfo struct {
	Error      *errBody `json:"error"`
	ServerTime struct {
		ServerTime int64 `json:"serverTime"`
	} `json:"serverTime"`
	ServerSetting struct {
		Error    *errBody `json:"error"`
		Settings struct {
			PaymentSettingV2 struct {
				OpenPayment      uint32 `json:"openPayment"`
				PaymentPlatforms []any  `json:"paymentPlatforms"`
			} `json:"paymentSettingV2"`
			NicknameSetting struct {
				Enable    uint32 `json:"enable"`
				Nicknames []any  `json:"nicknames"`
			} `json:"nicknameSetting"`
		} `json:"settings"`
	} `json:"serverSetting"`
	ClientValue struct{} `json:"clientValue"`
	FriendList  struct {
		Error   *errBody `json:"error"`
		Friends []any    `json:"friends"`
	} `json:"friendList"`
	FriendApplyList struct {
		Error   *errBody `json:"error"`
		Applies []any    `json:"applies"`
	} `json:"friendApplyList"`
	MailInfo struct {
		Error *errBody `json:"error"`
		Mails []any    `json:"mails"`
	} `json:"mailInfo"`
	CharacterInfo struct {
		Error            *errBody  `json:"error"`
		Characters       []charRPC `json:"characters"`
		Skins            []uint32  `json:"skins"`
		MainCharacterID  uint32    `json:"mainCharacterId"`
		CharacterSort    []uint32  `json:"characterSort"`
		HiddenCharacters []uint32  `json:"hiddenCharacters"`
	} `json:"characterInfo"`
	BagInfo struct {
		Error *errBody `json:"error"`
		Bag   struct {
			Items []any `json:"items"`
		} `json:"bag"`
	} `json:"bagInfo"`
	TitleList struct {
		Error     *errBody `json:"error"`
		TitleList []uint32 `json:"titleList"`
	} `json:"titleList"`
	ActivityData    struct{} `json:"activityData"`
	MonthTicketInfo struct{} `json:"monthTicketInfo"`
	Achievement     struct {
		Error      *errBody `json:"error"`
		Progresses []any    `json:"progresses"`
	} `json:"achievement"`
	AccountSettings struct {
		Error *errBody `json:"error"`
	} `json:"accountSettings"`
	AllCommonViews []any `json:"allCommonViews"`
}

// charRPC is a character entry within characterInfo.
type charRPC struct {
	CharID     uint32 `json:"charid"`
	Level      uint32 `json:"level"`
	Exp        uint32 `json:"exp"`
	Skin       uint32 `json:"skin"`
	IsUpgraded bool   `json:"isUpgraded"`
}

// notifyAccountUpdate pushes account numbers after login.
type notifyAccountUpdate struct {
	Update *updatePayload `json:"update"`
}

type updatePayload struct {
	Numerical []numEntry `json:"numerical"`
	Character struct {
		Characters      []charRPC `json:"characters"`
		Skins           []uint32  `json:"skins"`
		MainCharacterID uint32    `json:"mainCharacterId"`
	} `json:"character"`
}

type numEntry struct {
	ID    uint32 `json:"id"`
	Final uint32 `json:"final"`
}

// accountRPC projects an account for client rendering.
func (h *handler) accountRPC(home *user.Home) *accountRPC {
	return &accountRPC{
		AccountID:  uint32(home.Account.ID),
		Nickname:   home.Account.Nickname,
		AvatarID:   uint32(home.Account.AvatarID),
		Level:      &levelRPC{ID: uint32(home.Account.LevelID), Score: uint32(home.Account.LevelScore)},
		Level3:     &levelRPC{ID: 1001, Score: 0},
		VIP:        uint32(home.Account.VIP),
		Title:      uint32(home.Account.Title),
		LoginTime:  uint32(home.Account.LastLogin),
		LogoutTime: 0,
		RoomID:     0,
		AntiAddiction: struct {
			OnlineDuration uint32 `json:"onlineDuration"`
		}{},
		Email:       home.Account.Username,
		PhoneVerify: 0,
		EmailVerify: 1,
		AvatarFrame: 0,
		Gold:        uint32(home.Wallet.Gold),
		Diamond:     uint32(home.Wallet.Diamond),
		SkinTicket:  uint32(home.Wallet.SkinTicket),
		Signature:   home.Account.Signature,
		Verified:    uint32(home.Account.Verified),
	}
}

func (h *handler) charRPCs(home *user.Home) []charRPC {
	out := make([]charRPC, 0, len(home.Characters))
	for _, c := range home.Characters {
		out = append(out, charRPC{
			CharID:     uint32(c.CharID),
			Level:      uint32(c.Level),
			Exp:        uint32(c.Exp),
			Skin:       uint32(c.SkinID),
			IsUpgraded: false,
		})
	}
	return out
}

func characterSkins(home *user.Home) []uint32 {
	var out []uint32
	for _, c := range home.Characters {
		if c.SkinID != 0 {
			out = append(out, uint32(c.SkinID))
		}
	}
	return out
}

func (h *handler) charIDs(home *user.Home) []uint32 {
	out := make([]uint32, 0, len(home.Characters))
	for _, c := range home.Characters {
		out = append(out, uint32(c.CharID))
	}
	return out
}

// roomView projects a room aggregate for client rendering.
func (h *handler) roomView(r *room.Room) *roomView {
	v := &roomView{
		RoomID:         r.ID,
		OwnerID:        r.OwnerID,
		Mode:           map[string]any{"mode": r.Mode.Mode, "detailRule": r.Mode.DetailRule},
		MaxPlayerCount: r.MaxPlayerCount,
		IsPlaying:      r.GameStarted,
		PublicLive:     r.PublicLive,
		Positions:      []uint32{},
	}
	for i, p := range r.Players {
		v.Positions = append(v.Positions, r.ID*10+uint32(i))
		if p.Robot {
			v.Robots = append(v.Robots, h.playerView(&p))
			v.RobotCount++
			continue
		}
		v.Persons = append(v.Persons, h.playerView(&p))
		if p.Ready {
			v.ReadyList = append(v.ReadyList, p.AccountID)
		}
	}
	return v
}

func (h *handler) playerView(p *room.Player) playerGameView {
	v := playerGameView{
		AccountID: p.AccountID,
		AvatarID:  p.AvatarID,
		Nickname:  p.Nickname,
	}
	v.Level.ID = 1001
	v.Level3.ID = 1001
	v.Views = []any{}
	return v
}

func (h *handler) updatePayload(accountID int64) *updatePayload {
	home, err := h.user.Home(context.Background(), accountID)
	if err != nil {
		return &updatePayload{}
	}
	u := &updatePayload{}
	u.Numerical = []numEntry{
		{ID: 100001, Final: uint32(home.Wallet.Gold)},
		{ID: 100002, Final: uint32(home.Wallet.Diamond)},
		{ID: 100003, Final: uint32(home.Wallet.SkinTicket)},
		{ID: 100004, Final: uint32(home.Account.VIP)},
	}
	u.Character.Characters = h.charRPCs(home)
	u.Character.Skins = characterSkins(home)
	u.Character.MainCharacterID = 200001
	if len(home.Characters) > 0 {
		u.Character.MainCharacterID = uint32(home.Characters[0].CharID)
	}
	return u
}
