// Command simclient replays the Majsoul client's lobby and game-session RPC
// sequences against a local server and prints decoded frames, so protocol
// behavior can be verified and diffed against the reference implementation.
//
//	go run ./cmd/simclient -addr ws://127.0.0.1:8441
//	go run ./cmd/simclient -game ws://127.0.0.1:8443
package main

import (
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"

	"github.com/qy-info/gosoul/internal/protocol"
)

func main() {
	addr := flag.String("addr", "ws://127.0.0.1:8441", "lobby websocket")
	method := flag.String("method", "", "if set, print response then exit")
	login := flag.String("login", "test001", "account to login")
	pwd := flag.String("pwd", "c1c2539fc0479405008be63722e30b7aebc177a86418fcd2ecd69a2fdd515fcf", "client-side password hash")
	gameAddr := flag.String("game", "", "if set, dial the FastTest game port directly and verify auth")
	flag.Parse()

	if *gameAddr != "" {
		runGameMode(*gameAddr)
		return
	}

	u, err := url.Parse(*addr)
	if err != nil {
		log.Fatal(err)
	}
	u.Path = "/gateway"
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial: ", err)
	}
	defer c.Close()

	reg, err := protocol.Load()
	if err != nil {
		log.Fatal(err)
	}

	var msgID uint16
	send := func(name string, typeName string, v any) {
		msgID++
		var payload []byte
		if v != nil {
			payload, err = reg.EncodeAsDynamic(typeName, v)
			if err != nil {
				log.Fatalf("encode %s: %v", name, err)
			}
		}
		frame := protocol.EncodeRequestLE(msgID, name, payload)
		if err := c.WriteMessage(websocket.BinaryMessage, frame); err != nil {
			log.Fatal("write: ", err)
		}
	}

	recv := func(respType string) []byte {
		for {
			_, data, err := c.ReadMessage()
			if err != nil {
				log.Fatal("read: ", err)
			}
			f, err := protocol.DecodeFrame(data)
			if err != nil {
				log.Fatalf("decode: %v hex=%x", err, data)
			}
			got := ""
			if respType != "" && len(f.Data) > 0 {
				got = decodeJSON(reg, respType, f.Data)
			} else if f.Type == protocol.MsgNotify && f.Name != "" {
				got = decodeJSON(reg, f.Name, f.Data)
			}
			fmt.Printf(">>> frame type=%d name=%q dataLen=%d json=%s\nhex=%s\n", f.Type, f.Name, len(f.Data), got, hex.EncodeToString(data))
			if respType == "" || f.Type != protocol.MsgNotify {
				return data
			}
		}
	}

	send(protocol.MethodRouteRequestConnection, "lq.ReqRequestConnection", map[string]any{
		"type": 1, "routeId": "route-1", "timestamp": time.Now().Unix(),
	})
	recv("lq.ResRequestConnection")

	send(protocol.MethodLobbyOauth2Check, "lq.ReqOauth2Check", map[string]any{})
	recv(protocol.TypeResOauth2Check)

	send(protocol.MethodLobbyOauth2Signup, "lq.ReqOauth2Signup", map[string]any{"type": 0})
	recv(protocol.TypeResOauth2Signup)

	send(protocol.MethodLobbyOauth2Login, protocol.TypeReqOauth2Login, map[string]any{"accessToken": "local-token-x"})
	recv(protocol.TypeResLogin)

	send(protocol.MethodLobbyLogin, protocol.TypeReqLogin, map[string]any{
		"account":             *login,
		"password":            *pwd,
		"device":              map[string]any{"platform": "pc", "os": "mac", "isBrowser": true, "software": "Chrome", "salePlatform": "web", "screenType": 1},
		"clientVersion":       map[string]any{"resource": "0.16.272", "package": "4.0.46"},
		"currencyPlatforms":   []any{1, 2, 5, 6, 8, 10, 11},
		"genAccessToken":      true,
		"type":                0,
		"clientVersionString": "WebGL_2022-0.16.272",
		"tag":                 "cn",
	})
	recv(protocol.TypeResLogin)

	seq := [][2]string{
		{".lq.Lobby.loginBeat", "lq.ResCommon"},
		{".lq.Lobby.loginSuccess", "lq.ResCommon"},
		{".lq.Lobby.fetchAnnouncement", "lq.ResAnnouncement"},
		{".lq.Lobby.fetchInfo", "lq.ResFetchInfo"},
		{".lq.Lobby.fetchReviveCoinInfo", "lq.ResReviveCoinInfo"},
		{".lq.Lobby.fetchDailyTask", "lq.ResDailyTask"},
		{".lq.Lobby.fetchAchievementRate", "lq.ResFetchAchievementRate"},
		{".lq.Lobby.fetchCommentSetting", "lq.ResCommentSetting"},
		{".lq.Lobby.fetchRollingNotice", "lq.ResFetchRollingNotice"},
	}
	for _, step := range seq {
		send(step[0], "", nil)
		recv(step[1])
	}

	send(".lq.Lobby.createRoom", "lq.ReqCreateRoom", map[string]any{
		"playerCount": 4,
		"mode": map[string]any{
			"mode":       1,
			"detailRule": map[string]any{},
		},
		"clientVersionString": "WebGL_2022-0.16.272",
	})
	recv("lq.ResCreateRoom")

	send(".lq.Lobby.fetchRoom", "", nil)
	recv("lq.ResSelfRoom")

	if *method != "" {
		send(*method, "", nil)
		recv("lq.ResCommon")
	}
}

func decodeJSON(reg *protocol.Registry, name string, data []byte) string {
	m, err := reg.NewMessage(name)
	if err != nil {
		return "<" + err.Error() + ">"
	}
	if err := m.Unmarshal(data); err != nil {
		return "<unmarshal: " + err.Error() + ">"
	}
	b, err := m.MarshalJSON()
	if err != nil {
		return "<json: " + err.Error() + ">"
	}
	return string(b)
}

// runGameMode dials the FastTest game port and replays auth→enter→sync to
// verify the game-session RPC surface end to end.
func runGameMode(addr string) {
	u, err := url.Parse(addr)
	if err != nil {
		log.Fatal(err)
	}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial: ", err)
	}
	defer c.Close()

	reg, err := protocol.Load()
	if err != nil {
		log.Fatal(err)
	}
	var msgID uint16
	send := func(name, typeName string, v any) {
		msgID++
		payload, err := reg.EncodeAsDynamic(typeName, v)
		if err != nil {
			log.Fatalf("encode %s: %v", name, err)
		}
		frame := protocol.EncodeRequestLE(msgID, name, payload)
		if err := c.WriteMessage(websocket.BinaryMessage, frame); err != nil {
			log.Fatal("write: ", err)
		}
		for {
			_, data, err := c.ReadMessage()
			if err != nil {
				log.Fatal("read: ", err)
			}
			f, err := protocol.DecodeFrame(data)
			if err != nil {
				log.Fatalf("decode: %v hex=%x", err, data)
			}
			if f.Type == protocol.MsgNotify {
				got := ""
				if f.Name == protocol.ActionPrototypeNamespace {
					got = decodeAction(reg, f.Data)
				} else if len(f.Data) > 0 {
					got = decodeJSON(reg, f.Name, f.Data)
				}
				fmt.Printf(">>> notify name=%q dataLen=%d json=%s\n", f.Name, len(f.Data), got)
				continue
			}
			if f.Type != protocol.MsgResponse {
				continue
			}
			json := ""
			if len(f.Data) > 0 {
				json = decodeJSON(reg, respFor(name), f.Data)
			}
			fmt.Printf(">>> %s dataLen=%d json=%s\n", name, len(f.Data), json)
			break
		}
	}
	send(".lq.FastTest.authGame", "lq.ReqAuthGame", map[string]any{"accountId": 4, "token": "x", "gameUuid": "g1"})
	send(".lq.FastTest.enterGame", "lq.ReqCommon", map[string]any{})
	send(".lq.FastTest.syncGame", "lq.ReqSyncGame", map[string]any{"roundId": "", "step": 0})
	send(".lq.FastTest.confirmNewRound", "lq.ReqCommon", map[string]any{})
	send(".lq.FastTest.checkNetworkDelay", "lq.ReqCommon", map[string]any{})
	send(".lq.FastTest.inputOperation", "lq.ReqSelfOperation", map[string]any{"type": 0, "tile": "1m", "timeuse": 500})
}

func respFor(method string) string {
	switch method {
	case ".lq.FastTest.authGame":
		return "lq.ResAuthGame"
	case ".lq.FastTest.enterGame":
		return "lq.ResEnterGame"
	case ".lq.FastTest.syncGame":
		return "lq.ResSyncGame"
	default:
		return "lq.ResCommon"
	}
}

// decodeAction unpacks an ActionPrototype notify payload. The wrapper is
// {name, data(base64 of xor(payload)), step}: parse it, decode base64, XOR
// back, and report the inner action name plus hex of the recovered payload.
func decodeAction(reg *protocol.Registry, data []byte) string {
	// ActionPrototype is {step=1, name=2, data=3} where data is a base64 string
	// of the XOR'd inner payload (the reference treats it as a string even
	// though the proto declares bytes).
	name, b64, step, ok := parseActionProto(data)
	if !ok {
		return "<bad action proto>"
	}
	_ = step
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "<b64: " + err.Error() + ">"
	}
	decoded := protocol.XORCodec(raw)
	inner, err := reg.NewMessage("lq." + name)
	if err != nil {
		return name + " (no proto)"
	}
	if err := inner.Unmarshal(decoded); err != nil {
		return name + " (unmarshal: " + err.Error() + ")"
	}
	jb, _ := inner.MarshalJSON()
	return name + ": " + string(jb)
}

func parseActionProto(b []byte) (name, b64 string, step uint32, ok bool) {
	i := 0
	for i < len(b) {
		tag, n := rdTag(b[i:])
		if n <= 0 {
			return "", "", 0, false
		}
		i += n
		field := int(tag >> 3)
		switch field {
		case 1:
			v, m := rdVar(b[i:])
			if m <= 0 {
				return "", "", 0, false
			}
			step, i = uint32(v), i+m
		case 2:
			l, m := rdVar(b[i:])
			if m <= 0 || i+m+l > len(b) {
				return "", "", 0, false
			}
			name, i = string(b[i+m:i+m+l]), i+m+l
		case 3:
			l, m := rdVar(b[i:])
			if m <= 0 || i+m+l > len(b) {
				return "", "", 0, false
			}
			b64, i = string(b[i+m:i+m+l]), i+m+l
		default:
			_, m := rdVar(b[i:])
			if m <= 0 {
				return "", "", 0, false
			}
			i += m
		}
	}
	return name, b64, step, true
}

func rdTag(b []byte) (uint64, int) {
	var v uint64
	s := 0
	for i, c := range b {
		v |= uint64(c&0x7f) << uint(s)
		if c < 0x80 {
			return v, i + 1
		}
		s += 7
	}
	return 0, 0
}

func rdVar(b []byte) (int, int) {
	var v uint64
	s := 0
	for i, c := range b {
		v |= uint64(c&0x7f) << uint(s)
		if c < 0x80 {
			return int(v), i + 1
		}
		s += 7
	}
	return 0, 0
}
