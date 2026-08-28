// Command simclient replays a visitor oauth2 login against a Majsoul lobby
// WebSocket and prints the raw frames received, so a Go server's ResLogin can
// be diffed byte-for-byte against the reference implementation.
//
//	go run ./cmd/simclient -addr ws://127.0.0.1:8441
package main

import (
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
	flag.Parse()

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
		return data
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
