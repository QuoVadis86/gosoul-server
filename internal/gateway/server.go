package gateway

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/elazarl/goproxy"
)

// Config wires the proxy to its route targets.
type Config struct {
	ListenAddr   string
	CA           *CA
	Domains      []string
	LobbyAddr    string
	ResourceAddr string
}

// Server is the MITM proxy entry point, built on elazarl/goproxy. Players
// point their client at it (HTTP proxy / PAC); game-domain traffic is
// terminated and served locally, everything else passes through untouched.
type Server struct {
	cfg    Config
	proxy  *goproxy.ProxyHttpServer
	routes *RouteTable
	log    *slog.Logger
}

// NewServer builds the proxy, wiring MITM CA and domain routing.
func NewServer(cfg Config, log *slog.Logger) (*Server, error) {
	if cfg.CA == nil {
		return nil, errors.New("gateway: CA required")
	}
	s := &Server{cfg: cfg, log: log, routes: &RouteTable{Domains: cfg.Domains}}

	if err := s.cfg.CA.SetAsGoproxyGlobal(); err != nil {
		return nil, err
	}

	p := goproxy.NewProxyHttpServer()
	p.Verbose = false
	p.Logger = slogPrintf{log: log}

	hosts := make([]*regexp.Regexp, 0, len(cfg.Domains))
	for _, d := range cfg.Domains {
		hosts = append(hosts, regexp.MustCompile(`([.-]|^)`+regexp.QuoteMeta(d)+`(:\d+)?$`))
	}
	gameHosts := goproxy.ReqHostMatches(hosts...)

	p.OnRequest(gameHosts).HandleConnect(goproxy.AlwaysMitm)
	p.OnRequest(gameHosts).DoFunc(s.handleRouted)

	s.proxy = p
	return s, nil
}

// ListenAndServe runs the proxy until the context is cancelled.
func (s *Server) ListenAndServe(addr string) error {
	s.log.Info("gateway listening", "addr", addr)
	return http.ListenAndServe(addr, s.proxy)
}

// Handler exposes the proxy as an http.Handler (for tests and embedding).
func (s *Server) Handler() http.Handler { return s.proxy }

// Shutdown the proxy.
func (s *Server) Shutdown(context.Context) error { return nil }

// handleRouted answers intercepted game-domain requests. clientgate API paths
// are served locally; everything else passes through upstream by returning nil.
func (s *Server) handleRouted(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	host := SplitHost(req.Host)
	var payload []byte
	switch {
	case strings.Contains(req.URL.Path, "/routes"):
		payload = routesResponse(s.cfg.LobbyAddr)
	case strings.Contains(req.URL.Path, "/upgrade_info"):
		payload = upgradeResponse()
	case strings.Contains(req.URL.Path, "/announce_list"):
		payload = announceResponse()
	default:
		s.log.Debug("gateway passthrough", "host", host, "path", req.URL.Path)
		return req, nil
	}
	s.log.Info("gateway api answered", "host", host, "path", req.URL.Path)
	return req, respondJSON(payload)
}

func respondJSON(body []byte) *http.Response {
	return &http.Response{
		StatusCode:    http.StatusOK,
		Header:        http.Header{"Content-Type": {"application/json; charset=utf-8"}},
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
	}
}

// slogPrintf adapts slog to goproxy's Logger interface.
type slogPrintf struct{ log *slog.Logger }

func (s slogPrintf) Printf(format string, args ...any) {
	s.log.Debug(format, args...)
}
