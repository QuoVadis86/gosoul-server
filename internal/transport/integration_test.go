package transport

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/qy-info/gosoul/internal/lobby"
	"github.com/qy-info/gosoul/internal/protocol"
	"github.com/qy-info/gosoul/internal/room"
	"github.com/qy-info/gosoul/internal/router"
	"github.com/qy-info/gosoul/internal/storage"
	"github.com/qy-info/gosoul/internal/user"
)

func startLobbyServer(t *testing.T) (*protocol.Registry, *httptest.Server) {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	store, err := storage.Open(t.TempDir() + "/it.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	reg, err := protocol.Load()
	if err != nil {
		t.Fatal(err)
	}
	rtr := router.New(reg)
	svc := user.NewService(store.Account, store.Character, store.Wallet)
	lobby.Handlers(svc, log, rtr, reg, room.New(nil))

	server := New(rtr, reg, log)
	ts := httptest.NewServer(http.HandlerFunc(server.HandleHTTP))
	t.Cleanup(ts.Close)
	return reg, ts
}

// client is a minimal Majsoul protocol client for tests.
type client struct {
	t    *testing.T
	reg  *protocol.Registry
	conn *websocket.Conn
	seq  uint16
}

func dial(t *testing.T, reg *protocol.Registry, ts *httptest.Server) *client {
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return &client{t: t, reg: reg, conn: conn}
}

func (c *client) request(method, reqType string, v any) (respType string, respJSON string) {
	var payload []byte
	if reqType != "" {
		var err error
		payload, err = c.reg.EncodeAsDynamic(reqType, v)
		if err != nil {
			payload = nil
		}
	}
	c.seq++
	if err := c.conn.WriteMessage(websocket.BinaryMessage,
		protocol.EncodeRequestLE(c.seq, method, payload)); err != nil {
		c.t.Fatalf("write: %v", err)
	}

	for {
		_ = c.conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			c.t.Fatalf("read: %v", err)
		}
		frame, err := protocol.DecodeFrame(data)
		if err != nil {
			c.t.Fatalf("decode frame: %v", err)
		}
		if frame.Type == protocol.MsgRequest ||
			(frame.Type == protocol.MsgNotify && frame.Name != protocol.NotifyAccountUpdate) {
			continue
		}
		if frame.MsgID != c.seq {
			continue
		}
		route, _ := c.reg.RouteFor(method)
		msg, err := c.reg.NewMessage(route.RespType)
		if err != nil {
			c.t.Fatalf("new message: %v (%s)", err, method)
		}
		if err := msg.Unmarshal(frame.Data); err != nil {
			c.t.Fatalf("unmarshal resp: %v", err)
		}
		jsonBytes, _ := msg.ToJSON()
		return route.RespType, string(jsonBytes)
	}
}

type signupReq struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

type loginReq struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

type errRPC struct {
	Error *struct {
		Code int `json:"code"`
	} `json:"error"`
}

func signup(c *client, u, p string) error {
	_, body := c.request(protocol.MethodLobbySignup, protocol.TypeReqSignupAccount, &signupReq{Account: u, Password: p})
	var res errRPC
	_ = json.Unmarshal([]byte(body), &res)
	if res.Error != nil && res.Error.Code != 0 {
		return &rpcError{body: body}
	}
	return nil
}

func loginRB(c *client, u, p string) (uint32, error) {
	_, body := c.request(protocol.MethodLobbyLogin, protocol.TypeReqLogin, &loginReq{Account: u, Password: p})
	var res errRPC
	if err := json.Unmarshal([]byte(body), &res); err != nil {
		return 0, err
	}
	if res.Error != nil && res.Error.Code != 0 {
		return 0, &rpcError{body: body}
	}
	var login struct {
		AccountID uint32 `json:"accountId"`
	}
	_ = json.Unmarshal([]byte(body), &login)
	return login.AccountID, nil
}

type rpcError struct {
	body string
}

func (e *rpcError) Error() string { return e.body }

func TestLoginFlow(t *testing.T) {
	reg, ts := startLobbyServer(t)
	c := dial(t, reg, ts)

	c.request(protocol.MethodRouteRequestConnection, protocol.TypeReqCommon, struct{}{})
	c.request(protocol.MethodRouteHeartbeat, protocol.TypeReqCommon, struct{}{})
	c.request(protocol.MethodLobbyPrepareLogin, protocol.TypeReqCommon, struct{}{})

	if err := signup(c, "test010", "pw"); err != nil {
		t.Fatalf("signup: %v", err)
	}
	id, err := loginRB(c, "test010", "pw")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if id == 0 {
		t.Fatal("login returned no account id")
	}
}

func TestSignupThenLoginSameID(t *testing.T) {
	reg, ts := startLobbyServer(t)

	regOnce := dial(t, reg, ts)
	if err := signup(regOnce, "persist-me", "x"); err != nil {
		t.Fatal(err)
	}

	loginID := func() uint32 {
		c := dial(t, reg, ts)
		id, err := loginRB(c, "persist-me", "x")
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	id1 := loginID()
	id2 := loginID()
	if id1 != id2 {
		t.Fatalf("account id changed %d -> %d", id1, id2)
	}
}

func TestLoginUnknownAccountFails(t *testing.T) {
	reg, ts := startLobbyServer(t)
	c := dial(t, reg, ts)
	if _, err := loginRB(c, "ghost-user", "x"); err == nil {
		t.Fatal("unknown account must error")
	}
}

func TestLoginWrongPasswordFails(t *testing.T) {
	reg, ts := startLobbyServer(t)
	c := dial(t, reg, ts)
	if err := signup(c, "pwcheck", "right"); err != nil {
		t.Fatal(err)
	}
	if _, err := loginRB(c, "pwcheck", "wrong"); err == nil {
		t.Fatal("wrong password must error")
	}
}

func TestSignupDuplicateFails(t *testing.T) {
	reg, ts := startLobbyServer(t)
	c := dial(t, reg, ts)
	if err := signup(c, "dup", "a"); err != nil {
		t.Fatal(err)
	}
	if err := signup(c, "dup", "b"); err == nil {
		t.Fatal("duplicate username must error")
	}
}

func TestSurfaceMeetsFullRPCSurface(t *testing.T) {
	reg, ts := startLobbyServer(t)
	c := dial(t, reg, ts)

	sample := []string{
		protocol.MethodLobbyLogin,
		".lq.Lobby.createRoom",
		".lq.Lobby.startUnifiedMatch",
		".lq.Lobby.fetchMail",
		".lq.Lobby.readyRoom",
		".lq.FastTest.authGame",
		".lq.FastTest.syncGame",
	}
	known := map[string]bool{}
	for _, m := range reg.Methods() {
		known[m] = true
	}
	for _, method := range sample {
		if !known[method] {
			continue
		}
		route, _ := reg.RouteFor(method)
		respType, body := c.request(method, "", struct{}{})
		if respType != route.RespType {
			t.Errorf("%s: resp type %s, want %s", method, respType, route.RespType)
		}
		if body == "" {
			t.Errorf("%s: empty body", method)
		}
	}
}
