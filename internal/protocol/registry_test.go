package protocol

import "testing"

func TestRegistryLoad(t *testing.T) {
	reg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	route, ok := reg.RouteFor(".lq.Lobby.oauth2Login")
	if !ok {
		t.Fatalf(".lq.Lobby.oauth2Login not routed")
	}
	if route.ReqType != "lq.ReqOauth2Login" {
		t.Fatalf("ReqType = %s", route.ReqType)
	}
	if route.RespType != "lq.ResLogin" {
		t.Fatalf("RespType = %s (want lq.ResLogin)", route.RespType)
	}

	route2, ok := reg.RouteFor(".lq.FastTest.authGame")
	if !ok {
		t.Fatalf("authGame not routed")
	}
	if route2.ReqType != "lq.ReqAuthGame" || route2.RespType != "lq.ResAuthGame" {
		t.Fatalf("authGame types = %s/%s", route2.ReqType, route2.RespType)
	}

	if _, ok := reg.NotifyTypeFor(".lq.NotifyMatchGameStart"); !ok {
		t.Fatalf("NotifyMatchGameStart not registered")
	}
}

func TestRegistryNewMessage(t *testing.T) {
	reg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	m, err := reg.NewMessage("lq.ResHeartbeat")
	if err != nil {
		t.Fatalf("NewMessage: %v", err)
	}
	b, err := m.Marshal()
	if err != nil {
		t.Fatalf("Marshal empty ResHeartbeat: %v", err)
	}
	// An empty ResHeartbeat encodes to zero bytes.
	if len(b) != 0 {
		t.Fatalf("empty message encoded to %d bytes", len(b))
	}
}
