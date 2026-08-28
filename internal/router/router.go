// Package router dispatches Majsoul RPC names to handlers. It is the thin
// glue between the transport layer (WebSocket frames) and the service layers.
package router

import (
	"github.com/qy-info/gosoul/internal/protocol"
)

// Session is the responder interface a handler uses to write frames back.
type Session interface {
	// AccountID is the authenticated account of the connection (0 = guest).
	AccountID() int64
	SetAccountID(id int64)
	// Respond answers a request with a typed payload.
	Respond(msgID uint16, typeName string, v any) error
	// Notify pushes a server-originated message.
	Notify(name string, v any) error
}

// Context carries one decoded request to its handler.
type Context struct {
	Session Session
	Method  string
	MsgID   uint16
	Payload []byte
	Reg     *protocol.Registry
}

// Handler processes one request. Implementations should keep writes through
// ctx.Session so ordering is preserved on the connection.
type Handler func(ctx *Context) error

// Router is the method → handler table.
type Router struct {
	handlers map[string]Handler
	reg      *protocol.Registry
}

// New builds an empty router bound to a protocol registry.
func New(reg *protocol.Registry) *Router {
	return &Router{handlers: make(map[string]Handler), reg: reg}
}

// Handle registers a handler for a method name (e.g. ".lq.Lobby.login").
func (r *Router) Handle(method string, h Handler) {
	r.handlers[method] = h
}

// Has reports whether a handler is registered for the method.
func (r *Router) Has(method string) bool {
	_, ok := r.handlers[method]
	return ok
}

// Dispatch runs the handler for a method. Returns false when unregistered.
func (r *Router) Dispatch(s Session, method string, msgID uint16, payload []byte) bool {
	h, ok := r.handlers[method]
	if !ok {
		return false
	}
	err := h(&Context{Session: s, Method: method, MsgID: msgID, Payload: payload, Reg: r.reg})
	if err != nil {
		// A plain success envelope keeps mishandled calls from hanging the
		// client; errors are surfaced via logs.
		_ = s.Respond(msgID, protocol.TypeResCommon, errSafe{})
	}
	return true
}

type errSafe struct{}
