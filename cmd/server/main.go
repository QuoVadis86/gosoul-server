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
	"github.com/qy-info/gosoul/internal/game"
	"github.com/qy-info/gosoul/internal/gateway"
	"github.com/qy-info/gosoul/internal/lobby"
	"github.com/qy-info/gosoul/internal/paipu"
	"github.com/qy-info/gosoul/internal/protocol"
	"github.com/qy-info/gosoul/internal/room"
	"github.com/qy-info/gosoul/internal/router"
	"github.com/qy-info/gosoul/internal/storage"
	"github.com/qy-info/gosoul/internal/transport"
	"github.com/qy-info/gosoul/internal/user"
)

// roomAccounts adapts the user service to the room domain's account lookup.
type roomAccounts struct{ svc *user.Service }

func (a roomAccounts) Account(accountID uint32) (string, uint32) {
	acc, err := a.svc.Get(context.Background(), int64(accountID))
	if err != nil {
		return "Player", 400101
	}
	return acc.Nickname, uint32(acc.AvatarID)
}

// achHook advances achievement counters when a round finishes.
type achHook struct{ svc *user.Service }

func (h achHook) Increment(ctx context.Context, accountID int64, achieveID, delta int64) error {
	cur, err := h.svc.Achievements(ctx, accountID)
	if err != nil {
		return err
	}
	progress := delta
	for _, a := range cur {
		if a.AchieveID == achieveID {
			progress += a.Progress
			break
		}
	}
	return h.svc.SetAchievement(ctx, user.Achievement{AccountID: accountID, AchieveID: achieveID, Progress: progress})
}

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

	svc := user.NewService(store.Account, store.Character, store.Wallet, store.Achieve)

	reg, err := protocol.Load()
	if err != nil {
		slog.Error("protocol", "err", err)
		os.Exit(1)
	}
	r := router.New(reg)
	roomSvc := room.New(roomAccounts{svc})
	lobby.Handlers(svc, log, r, reg, roomSvc, cfg.Game.Addr())
	transportSrv := transport.New(r, reg, log)
	lobbyHTTP := &http.Server{Addr: cfg.Lobby.Addr(), Handler: http.HandlerFunc(transportSrv.HandleHTTP)}

	gameRouter := router.New(reg)
	ppSvc := paipu.New(store.Paipu)
	game.Handlers(gameRouter, log, ppSvc, achHook{svc})
	gameSrv := transport.New(gameRouter, reg, log)
	gameHTTP := &http.Server{Addr: cfg.Game.Addr(), Handler: http.HandlerFunc(gameSrv.HandleHTTP)}

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

	adm := admin.New(log, svc)
	adminSrv := &http.Server{Addr: cfg.Admin.Listen, Handler: adm.Handler()}

	errCh := make(chan error, 4)
	go func() { errCh <- gw.ListenAndServe(cfg.Gateway.Listen) }()
	go func() { errCh <- adminSrv.ListenAndServe() }()
	go func() { errCh <- lobbyHTTP.ListenAndServe() }()
	go func() { errCh <- gameHTTP.ListenAndServe() }()
	if httpsAddr := os.Getenv("GOSOUL_HTTPS_ADDR"); httpsAddr != "" {
		go func() { errCh <- gw.ListenAndServeTLS(httpsAddr) }()
	}

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
	_ = gameHTTP.Shutdown(shutdown)
	slog.Info("gosoul stopped")
}
