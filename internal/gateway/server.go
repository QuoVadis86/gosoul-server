package gateway

import (
	"bytes"
	"context"
	"crypto/tls"
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
	client *http.Client
	log    *slog.Logger
}

// NewServer builds the proxy, wiring MITM CA and domain routing.
func NewServer(cfg Config, log *slog.Logger) (*Server, error) {
	if cfg.CA == nil {
		return nil, errors.New("gateway: CA required")
	}
	s := &Server{
		cfg: cfg, log: log, routes: &RouteTable{Domains: cfg.Domains},
		client: &http.Client{Transport: &http.Transport{}}}

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

// handleRouted answers intercepted game-domain requests in proxy mode.
func (s *Server) handleRouted(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	if req.Method == http.MethodOptions {
		return req, &http.Response{
			StatusCode:    http.StatusOK,
			Header:        corsHeaders(),
			Body:          io.NopCloser(bytes.NewReader(nil)),
			ContentLength: 0,
		}
	}
	if res := s.answerLocal(req); res != nil {
		return req, res
	}
	s.log.Debug("gateway passthrough", "host", SplitHost(req.Host), "path", req.URL.Path)
	return req, nil
}

// answerLocal serves the clientgate API family locally; nil when the path is
// not one we own (then it is passed through).
func (s *Server) answerLocal(req *http.Request) *http.Response {
	var payload []byte
	switch {
	case strings.Contains(req.URL.Path, "/routes"):
		payload = routesResponse(s.cfg.LobbyAddr)
	case strings.Contains(req.URL.Path, "/upgrade_info"):
		payload = upgradeResponse()
	case strings.Contains(req.URL.Path, "/announce_list"):
		payload = announceResponse()
	default:
		return nil
	}
	s.log.Info("gateway api answered", "host", SplitHost(req.Host), "path", req.URL.Path)
	return respondJSON(payload)
}

// corsWrite answers a cross-origin preflight with the allowed headers.
func (s *Server) corsWrite(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodOptions {
		return false
	}
	h := w.Header()
	for _, k := range []string{"Access-Control-Allow-Origin", "Access-Control-Allow-Methods", "Access-Control-Allow-Headers"} {
		h.Set(k, corsHeaders().Get(k))
	}
	w.WriteHeader(http.StatusOK)
	return true
}

// HTTPSHandler serves intercepted game domains over a TLS listener directly
// (hosts/DNS-direct deployments, no proxy configuration needed).
func (s *Server) HTTPSHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.corsWrite(w, r) {
			return
		}
		if res := s.answerLocal(r); res != nil {
			defer res.Body.Close()
			for k, vs := range res.Header {
				for _, v := range vs {
					w.Header().Add(k, v)
				}
			}
			w.WriteHeader(res.StatusCode)
			io.Copy(w, res.Body)
			return
		}
		s.forward(w, r)
	})
}

// forward relays r to its original host.
func (s *Server) forward(w http.ResponseWriter, r *http.Request) {
	req := r.Clone(r.Context())
	req.RequestURI = ""
	req.URL.Scheme = "https"
	resp, err := s.client.Do(req)
	if err != nil {
		http.Error(w, "gateway: upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// ListenAndServeTLS runs the direct-HTTPS listener on addr. Certificates are
// issued per SNI hostname from the gateway CA, so hosts entries (or DNS)
// pointing game domains at this listener verify cleanly.
func (s *Server) ListenAndServeTLS(addr string) error {
	tlsCfg := &tls.Config{
		GetCertificate: func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
			cert, err := s.cfg.CA.CertFor(chi.ServerName)
			if err != nil {
				return nil, err
			}
			return cert, nil
		},
	}
	srv := &http.Server{Addr: addr, Handler: s.HTTPSHandler(), TLSConfig: tlsCfg}
	s.log.Info("gateway https listening", "addr", addr)
	return srv.ListenAndServeTLS("", "")
}

// corsHeaders allow the web client (loaded from game.maj-soul.com) to fetch
// the clientgate API across origins.
func corsHeaders() http.Header {
	return http.Header{
		"Content-Type":                 {"application/json; charset=utf-8"},
		"Access-Control-Allow-Origin":  {"*"},
		"Access-Control-Allow-Methods": {"GET, POST, OPTIONS"},
		"Access-Control-Allow-Headers": {"Content-Type"},
	}
}

func respondJSON(body []byte) *http.Response {
	return &http.Response{
		StatusCode:    http.StatusOK,
		Header:        corsHeaders(),
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
	}
}

// slogPrintf adapts slog to goproxy's Logger interface.
type slogPrintf struct{ log *slog.Logger }

func (s slogPrintf) Printf(format string, args ...any) {
	s.log.Debug(format, args...)
}
