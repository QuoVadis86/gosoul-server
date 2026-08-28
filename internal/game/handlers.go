package game

import (
	"context"
	"fmt"
	"time"

	"github.com/qy-info/gosoul/internal/game/engine"
	"github.com/qy-info/gosoul/internal/router"
)

func (h *handlers) authGame(ctx *router.Context) error {
	var req struct {
		AccountID uint32 `json:"accountId"`
		Token     string `json:"token"`
		GameID    string `json:"gameUuid"`
	}
	if err := ctx.Reg.DecodeInto("lq.ReqAuthGame", ctx.Payload, &req); err != nil {
		return err
	}
	accountID := req.AccountID
	if accountID == 0 {
		accountID = uint32(ctx.Session.AccountID())
	}
	if accountID == 0 {
		accountID = 10001
	}
	seat := 0
	s := &session{
		AccountID: accountID,
		Seat:      seat,
		GameUUID:  req.GameID,
		StartedAt: time.Now(),
		paipu:     h.paipu,
	}
	if s.GameUUID == "" {
		s.GameUUID = fmt.Sprintf("game-%d-%d", accountID, time.Now().Unix())
	}
	h.sessions[ctx.Session] = s
	h.log.Info("game auth", "account", accountID, "uuid", s.GameUUID)

	seatList := []uint32{accountID, 10001, 10002, 10003}
	players := []playerGameView{seatView(accountID, "LocalPlayer")}
	robots := []playerGameView{
		seatView(10001, "AI_1"),
		seatView(10002, "AI_2"),
		seatView(10003, "AI_3"),
	}

	return ctx.Session.Respond(ctx.MsgID, "lq.ResAuthGame", &resAuthGame{
		Error:       errorCode(0),
		SeatList:    seatList,
		Players:     players,
		Robots:      robots,
		IsGameStart: true,
		GameConfig: gameConfig{
			Category: 1,
			Meta:     gameMetaData{ModeID: 2, RoomID: 1},
			Mode:     gameMode{Mode: 1},
		},
		ReadyIDList: seatList,
	})
}

func (h *handlers) enterGame(ctx *router.Context) error {
	if err := ctx.Session.Respond(ctx.MsgID, "lq.ResEnterGame", &resEnterGame{
		Error: errorCode(0),
		IsEnd: false,
		Step:  0,
	}); err != nil {
		return err
	}
	s := h.sessions[ctx.Session]
	if s == nil || s.round != nil {
		return nil
	}
	return h.startRound(context.Background(), s, ctx.Session)
}

func (h *handlers) syncGame(ctx *router.Context) error {
	return ctx.Session.Respond(ctx.MsgID, "lq.ResSyncGame", &resSyncGame{
		Error: errorCode(0),
		IsEnd: false,
		Step:  0,
	})
}

func (h *handlers) confirmNewRound(ctx *router.Context) error {
	return h.ok(ctx)
}

func (h *handlers) inputOperation(ctx *router.Context) error {
	var req struct {
		Type      uint32 `json:"type"`
		Tile      string `json:"tile"`
		Cancel    bool   `json:"cancelOperation"`
		TimeUse   uint32 `json:"timeuse"`
		TileState int32  `json:"tileState"`
	}
	if err := ctx.Reg.DecodeInto("lq.ReqSelfOperation", ctx.Payload, &req); err != nil {
		return err
	}
	s := h.sessions[ctx.Session]
	if s == nil || s.round == nil || s.round.drv == nil {
		return h.ok(ctx)
	}
	op := humanOp{Type: req.Type, Tile: engine.Tile(req.Tile), Cancel: req.Cancel, TimeUse: req.TimeUse, TileState: req.TileState}
	if err := s.round.drv.Deliver(s.Seat, op); err != nil {
		// Drive already complete; still answer ok to keep the client moving.
		return h.ok(ctx)
	}
	return h.ok(ctx)
}

func (h *handlers) inputCPG(ctx *router.Context) error {
	return h.ok(ctx)
}

func seatView(accountID uint32, nickname string) playerGameView {
	return playerGameView{
		AccountID: accountID,
		AvatarID:  400101,
		Nickname:  nickname,
		Level:     accountLevel{ID: 1001},
		Level3:    accountLevel{ID: 1001},
		Character: character{CharID: 200001, Level: 1},
		Views:     []any{},
	}
}
