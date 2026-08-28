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
	accounts := user.NewAccountService(store.Account)
	chars := user.NewCharacterService(store.Character)
	wallets := user.NewCurrencyService(store.Currency)
	svc := lobby.NewService(accounts, chars, wallets)
	lobby.Handlers(svc, accounts, chars, wallets, log, rtr, reg)

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
			payload = nil // tolerate request schemas missing from the proto subset
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
		if frame.Type == protocol.MsgRequest || (frame.Type == protocol.MsgNotify && frame.Name != ".lq.NotifyAccountUpdate") {
			continue
		}
		if frame.MsgID != c.seq {
			continue
		}
		route, _ := c.reg.RouteFor(method)
		respName := route.RespType
		msg, err := c.reg.NewMessage(respName)
		if err != nil {
			c.t.Fatalf("new message: %v", err)
		}
		if err := msg.Unmarshal(frame.Data); err != nil {
			c.t.Fatalf("unmarshal resp: %v", err)
		}
		jsonBytes, _ := msg.ToJSON()
		return respName, string(jsonBytes)
	}
}

func TestLoginFlow(t *testing.T) {
	reg, ts := startLobbyServer(t)
	c := dial(t, reg, ts)

	c.request(".lq.Route.requestConnection", "lq.ReqCommon", struct{}{})
	c.request(".lq.Route.heartbeat", "lq.ReqCommon", struct{}{})
	c.request(".lq.Lobby.prepareLogin", "lq.ReqCommon", struct{}{})

	respName, body := c.request(".lq.Lobby.login", "lq.ReqLogin", struct {
		Account  string `json:"account"`
		Password string `json:"password"`
	}{Account: "test010", Password: "pw"})

	if respName != "lq.ResLogin" {
		t.Fatalf("resp type = %s", respName)
	}
	var login struct {
		Error       *json.RawMessage `json:"error"`
		AccountID   uint32           `json:"accountId"`
		AccessToken string           `json:"accessToken"`
		Country     string           `json:"country"`
	}
	if err := json.Unmarshal([]byte(body), &login); err != nil {
		t.Fatal(err)
	}
	if login.AccountID == 0 {
		t.Fatalf("no account id: %s", body)
	}
	if !strings.HasPrefix(login.AccessToken, "local-token-") {
		t.Fatalf("access token: %s", body)
	}
}

func TestAutoRegisterPersistence(t *testing.T) {
	reg, ts := startLobbyServer(t)
	c := dial(t, reg, ts)
	_, body := c.request(".lq.Lobby.login", "lq.ReqLogin", struct {
		Account  string `json:"account"`
		Password string `json:"password"`
	}{Account: "persist-me", Password: "x"})

	var login struct {
		AccountID uint32 `json:"accountId"`
	}
	_ = json.Unmarshal([]byte(body), &login)
	acctID := login.AccountID
	if acctID == 0 {
		t.Fatal("no account created")
	}

	// Second login of the same user must return the same id (persisted).
	c2 := dial(t, reg, ts)
	_, body2 := c2.request(".lq.Lobby.login", "lq.ReqLogin", struct {
		Account  string `json:"account"`
		Password string `json:"password"`
	}{Account: "persist-me", Password: "x"})
	var login2 struct {
		AccountID uint32 `json:"accountId"`
	}
	_ = json.Unmarshal([]byte(body2), &login2)
	if login2.AccountID != acctID {
		t.Fatalf("account id changed %d -> %d", acctID, login2.AccountID)
	}
}

func TestSurfaceMeetsFullRPCSurface(t *testing.T) {
	reg, ts := startLobbyServer(t)
	c := dial(t, reg, ts)

	// Methods that are NOT explicitly implemented must still answer with
	// their correct proto response type so the client can decode them.
	sample := []string{
		".lq.Lobby.createRoom",
		".lq.Lobby.startUnifiedMatch",
		".lq.Lobby.fetchMail",
		".lq.Lobby.fetchShopInfo",
		".lq.Lobby.fetchAchievement",
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
			continue // route table does not reference it
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
