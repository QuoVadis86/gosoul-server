package lobby

import (
	"context"
	"fmt"
	"time"
)

func now() int64                         { return time.Now().Unix() }
func accessToken(accountID int64) string { return fmt.Sprintf("local-token-%d", accountID) }

// empty is the generic success envelope.
type empty struct{}

// ResRequestConnection: result/timestamp.
type resRequestConnection struct {
	Error     *errBody `json:"error"`
	Timestamp uint32   `json:"timestamp"`
	Result    uint32   `json:"result"`
}

type resHeartbeat struct {
	Error *errBody `json:"error"`
}

type errBody struct{}

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

// accountRPC is the nested Account projection the client renders.
type accountRPC struct {
	AccountID  uint32 `json:"accountId"`
	Nickname   string `json:"nickname"`
	AvatarID   uint32 `json:"avatarId"`
	Gold       uint32 `json:"gold"`
	Diamond    uint32 `json:"diamond"`
	SkinTicket uint32 `json:"skinTicket"`
	VIP        uint32 `json:"vip"`
	Title      uint32 `json:"title"`
	Signature  string `json:"signature"`
	Verified   uint32 `json:"verified"`
	Email      string `json:"email"`
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
		Error           *errBody  `json:"error"`
		Characters      []charRPC `json:"characters"`
		Skins           []uint32  `json:"skins"`
		MainCharacterID uint32    `json:"mainCharacterId"`
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
func (h *handler) accountRPC(state *AccountState) *accountRPC {
	return &accountRPC{
		AccountID:  uint32(state.Account.ID),
		Nickname:   state.Account.Nickname,
		AvatarID:   uint32(state.Account.AvatarID),
		Gold:       uint32(state.Gold),
		Diamond:    uint32(state.Diamond),
		SkinTicket: uint32(state.SkinTicket),
		VIP:        uint32(state.Account.VIP),
		Title:      uint32(state.Account.Title),
		Signature:  state.Account.Signature,
		Verified:   uint32(state.Account.Verified),
		Email:      state.Account.Username,
	}
}

func (h *handler) charRPCs(state *AccountState) []charRPC {
	out := make([]charRPC, 0, len(state.Characters))
	for _, c := range state.Characters {
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

func characterSkins(state *AccountState) []uint32 {
	var out []uint32
	for _, c := range state.Characters {
		if c.SkinID != 0 {
			out = append(out, uint32(c.SkinID))
		}
	}
	return out
}

func (h *handler) updatePayload(accountID int64) *updatePayload {
	state, err := h.svc.Home(context.Background(), accountID)
	if err != nil {
		return &updatePayload{}
	}
	u := &updatePayload{}
	u.Numerical = []numEntry{
		{ID: 100001, Final: uint32(state.Gold)},
		{ID: 100002, Final: uint32(state.Diamond)},
		{ID: 100003, Final: uint32(state.SkinTicket)},
		{ID: 100004, Final: uint32(state.Account.VIP)},
	}
	u.Character.Characters = h.charRPCs(state)
	u.Character.Skins = characterSkins(state)
	if len(state.Characters) > 0 {
		u.Character.MainCharacterID = uint32(state.Characters[0].CharID)
	}
	return u
}
