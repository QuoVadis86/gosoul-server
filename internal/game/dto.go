package game

type errBody struct {
	Code uint32 `json:"code"`
}

func errorCode(code uint32) *errBody {
	if code == 0 {
		return &errBody{}
	}
	return &errBody{Code: code}
}

// resCommon is the generic success envelope.
type resCommon struct {
	Error *errBody `json:"error"`
}

// accountLevel mirrors lq.AccountLevel {id,score}.
type accountLevel struct {
	ID    uint32 `json:"id"`
	Score uint32 `json:"score"`
}

// character mirrors the minimal lq.Character for the seat list.
type character struct {
	CharID uint32 `json:"charid"`
	Level  uint32 `json:"level"`
	Exp    uint32 `json:"exp"`
}

// playerGameView is a seat entry in ResAuthGame.
type playerGameView struct {
	AccountID uint32       `json:"accountId"`
	AvatarID  uint32       `json:"avatarId"`
	Nickname  string       `json:"nickname"`
	Level     accountLevel `json:"level"`
	Level3    accountLevel `json:"level3"`
	Character character    `json:"character"`
	Views     []any        `json:"views"`
}

// gameMode mirrors lq.GameMode for the auth config.
type gameMode struct {
	Mode uint32 `json:"mode"`
}

// gameMetaData mirrors lq.GameMetaData {modeId,roomId}.
type gameMetaData struct {
	ModeID uint32 `json:"modeId"`
	RoomID uint32 `json:"roomId"`
}

// gameConfig mirrors lq.GameConfig.
type gameConfig struct {
	Category uint32       `json:"category"`
	Meta     gameMetaData `json:"meta"`
	Mode     gameMode     `json:"mode"`
}

// resAuthGame answers .lq.FastTest.authGame.
type resAuthGame struct {
	Error       *errBody         `json:"error"`
	Players     []playerGameView `json:"players"`
	SeatList    []uint32         `json:"seatList"`
	IsGameStart bool             `json:"isGameStart"`
	GameConfig  gameConfig       `json:"gameConfig"`
	ReadyIDList []uint32         `json:"readyIdList"`
	Robots      []playerGameView `json:"robots"`
}

// actionProto mirrors the ActionPrototype wrapper pushed to clients.
type actionProto struct {
	Step uint32 `json:"step"`
	Name string `json:"name"`
	Data any    `json:"data"`
}

// gameRestore answers sync/enter when a round is live.
type gameRestore struct {
	Snapshot          any           `json:"snapshot"`
	Actions           []actionProto `json:"actions"`
	PassedWaitingTime uint32        `json:"passedWaitingTime"`
	GameState         uint32        `json:"gameState"`
	StartTime         uint32        `json:"startTime"`
}

// resEnterGame answers .lq.FastTest.enterGame.
type resEnterGame struct {
	Error       *errBody     `json:"error"`
	IsEnd       bool         `json:"isEnd"`
	Step        uint32       `json:"step"`
	GameRestore *gameRestore `json:"gameRestore"`
}

// resSyncGame answers .lq.FastTest.syncGame.
type resSyncGame struct {
	Error       *errBody     `json:"error"`
	IsEnd       bool         `json:"isEnd"`
	Step        uint32       `json:"step"`
	GameRestore *gameRestore `json:"gameRestore"`
}

// actionDealTile mirrors ActionDealTile.
type actionDealTile struct {
	Seat          uint32   `json:"seat"`
	Tile          string   `json:"tile"`
	LeftTileCount uint32   `json:"leftTileCount"`
	Doras         []string `json:"doras"`
}

// actionDiscardTile mirrors ActionDiscardTile.
type actionDiscardTile struct {
	Seat     uint32   `json:"seat"`
	Tile     string   `json:"tile"`
	Doras    []string `json:"doras"`
	Scores   []int    `json:"scores"`
	Liqibang uint32   `json:"liqibang"`
}

// huInfo is one winner inside ActionHule.
type huInfo struct {
	Seat  uint32   `json:"seat"`
	Zimo  bool     `json:"zimo"`
	Count uint32   `json:"count"`
	Fu    uint32   `json:"fu"`
	Title string   `json:"title"`
	Doras []string `json:"doras"`
}

// actionHu mirrors ActionHule.
type actionHu struct {
	Hules  []huInfo `json:"hules"`
	Scores []int    `json:"scores"`
}

// actionLiuJu mirrors ActionLiuJu.
type actionLiuJu struct {
	Type uint32 `json:"type"`
}
