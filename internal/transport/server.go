// Package transport owns the Majsoul WebSocket connection lifecycle: frame
// framing/unframing via protocol, and request dispatch to the router.
package transport

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/qy-info/gosoul/internal/protocol"
	"github.com/qy-info/gosoul/internal/router"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 50 * time.Second
)

// Server accepts client connections on a plain TCP port (the Majsoul Lobby
// gateway speaks ws, not wss).
type Server struct {
	router *router.Router
	reg    *protocol.Registry
	log    *slog.Logger
	up     *websocket.Upgrader
}

// New builds a transport server bound to a router.
func New(r *router.Router, reg *protocol.Registry, log *slog.Logger) *Server {
	return &Server{
		router: r,
		reg:    reg,
		log:    log,
		up: &websocket.Upgrader{
			CheckOrigin: func(*http.Request) bool { return true },
		},
	}
}

// HandleHTTP upgrades an incoming WebSocket request (usable with
// http.Server via Handler()).
func (s *Server) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	ws, err := s.up.Upgrade(w, r, nil)
	if err != nil {
		s.log.Debug("transport upgrade failed", "err", err)
		return
	}
	s.serve(newSession(ws, s.reg, s.router, s.log))
}

// serve drives one upgraded connection until it drops.
func (s *Server) serve(sess *Session) {
	s.log.Info("transport client connected", "remote", sess.ws.RemoteAddr())
	defer func() {
		s.log.Info("transport client disconnected", "remote", sess.ws.RemoteAddr())
		sess.ws.Close()
	}()

	_ = sess.ws.SetReadDeadline(time.Now().Add(pongWait))
	sess.ws.SetPongHandler(func(string) error {
		return sess.ws.SetReadDeadline(time.Now().Add(pongWait))
	})

	go sess.pingLoop()

	for {
		_, data, err := sess.ws.ReadMessage()
		if err != nil {
			s.log.Error("read loop exit", "err", err)
			return
		}
		sess.handleFrame(data)
	}
}

// Session wires one WebSocket to the RPC router.
type Session struct {
	ws        *websocket.Conn
	reg       *protocol.Registry
	router    *router.Router
	log       *slog.Logger
	mu        sync.Mutex
	accountID int64
}

func newSession(ws *websocket.Conn, reg *protocol.Registry, r *router.Router, log *slog.Logger) *Session {
	return &Session{ws: ws, reg: reg, router: r, log: log}
}

// AccountID implements router.Session.
func (s *Session) AccountID() int64 { return s.accountID }

// SetAccountID implements router.Session.
func (s *Session) SetAccountID(id int64) { s.accountID = id }

// Respond implements router.Session.
func (s *Session) Respond(msgID uint16, typeName string, v any) error {
	payload, err := s.reg.EncodeAsDynamic(typeName, v)
	if err != nil {
		s.log.Error("transport encode", "type", typeName, "err", err)
		return err
	}
	return s.write(protocol.EncodeResponse(msgID, payload))
}

// Notify implements router.Session.
func (s *Session) Notify(name string, v any) error {
	// A notify name ".lq.X" resolves to message type "lq.X".
	typeName, ok := s.reg.NotifyTypeFor(name)
	if !ok {
		return nil
	}
	payload, err := s.reg.EncodeAsDynamic(typeName, v)
	if err != nil {
		s.log.Error("transport notify encode", "type", typeName, "err", err)
		return err
	}
	return s.write(protocol.EncodeNotify(name, payload))
}

// ActionNotify pushes a battle action (XOR-wrapped ActionPrototype).
func (s *Session) ActionNotify(action string, v any, step uint32) error {
	var payload []byte
	if v == nil {
		payload = nil
	} else {
		actionType := "lq." + action
		var err error
		payload, err = s.reg.EncodeAsDynamic(actionType, v)
		if err != nil {
			return err
		}
	}
	return s.write(protocol.EncodeActionNotify(action, payload, step))
}

func (s *Session) handleFrame(data []byte) {
	frame, err := protocol.DecodeFrame(data)
	if err != nil {
		s.log.Debug("transport bad frame", "err", err)
		return
	}
	switch frame.Type {
	case protocol.MsgRequest:
		if !s.router.Dispatch(s, frame.Name, frame.MsgID, frame.Data) {
			s.log.Debug("transport unhandled method", "method", frame.Name)
			_ = s.Respond(frame.MsgID, "lq.ResCommon", empty{})
		}
	case protocol.MsgResponse:
		// client acknowledgements to our notifies: ignore
	}
}

func (s *Session) write(b []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.ws.SetWriteDeadline(time.Now().Add(writeWait))
	err := s.ws.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		s.log.Error("write failed", "err", err)
	}
	return err
}

func (s *Session) pingLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		err := s.ws.WriteMessage(websocket.PingMessage, nil)
		s.mu.Unlock()
		if err != nil {
			return
		}
	}
}

// empty is the minimal success envelope for unknown methods.
type empty struct{}
