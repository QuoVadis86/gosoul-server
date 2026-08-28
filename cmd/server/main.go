package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qy-info/gosoul/internal/admin"
	"github.com/qy-info/gosoul/internal/config"
	"github.com/qy-info/gosoul/internal/gateway"
	"github.com/qy-info/gosoul/internal/lobby"
	"github.com/qy-info/gosoul/internal/protocol"
	"github.com/qy-info/gosoul/internal/router"
	"github.com/qy-info/gosoul/internal/storage"
	"github.com/qy-info/gosoul/internal/transport"
	"github.com/qy-info/gosoul/internal/user"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(log)

	cfg, err := config.Load(os.Getenv("GOSOUL_CONFIG"))
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	store, err := storage.Open(cfg.Storage.Path)
	if err != nil {
		slog.Error("storage", "err", err, "path", cfg.Storage.Path)
		os.Exit(1)
	}
	defer store.Close()

	accounts := user.NewAccountService(store.Account)
	chars := user.NewCharacterService(store.Character)
	wallets := user.NewCurrencyService(store.Currency)

	reg, err := protocol.Load()
	if err != nil {
		slog.Error("protocol", "err", err)
		os.Exit(1)
	}
	r := router.New(reg)
	svc := lobby.NewService(accounts, chars, wallets)
	lobby.Handlers(svc, accounts, chars, wallets, log, r)
	transportSrv := transport.New(r, reg, log)
	lobbyHTTP := &http.Server{Addr: cfg.Lobby.Addr(), Handler: http.HandlerFunc(transportSrv.HandleHTTP)}

	gauc, err := gateway.LoadOrCreateCA(cfg.Gateway.CA.Cert, cfg.Gateway.CA.Key)
	if err != nil {
		slog.Error("gateway ca", "err", err)
		os.Exit(1)
	}
	gw, err := gateway.NewServer(gateway.Config{
		CA:           gauc,
		Domains:      gateway.DefaultDomains,
		LobbyAddr:    cfg.Lobby.Addr(),
		ResourceAddr: cfg.Resource.Addr(),
	}, log)
	if err != nil {
		slog.Error("gateway", "err", err)
		os.Exit(1)
	}

	adm := admin.New(log, accounts, chars, wallets)
	adminSrv := &http.Server{Addr: cfg.Admin.Listen, Handler: adm.Handler()}

	errCh := make(chan error, 2)
	go func() { errCh <- gw.ListenAndServe(cfg.Gateway.Listen) }()
	go func() { errCh <- adminSrv.ListenAndServe() }()
	go func() { errCh <- lobbyHTTP.ListenAndServe() }()

	slog.Info("gosoul starting",
		"gateway", cfg.Gateway.Listen,
		"lobby", cfg.Lobby.Addr(),
		"admin", cfg.Admin.Listen,
		"db", cfg.Storage.Path,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case e := <-errCh:
		slog.Error("server exited", "err", e)
		os.Exit(1)
	case <-ctx.Done():
	}

	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = gw.Shutdown(shutdown)
	_ = adminSrv.Shutdown(shutdown)
	_ = lobbyHTTP.Shutdown(shutdown)
	slog.Info("gosoul stopped")
}
