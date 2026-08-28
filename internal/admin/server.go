// Package admin is the operator-facing management surface (API only, no UI).
// It exposes account administration and resource grants on top of the user
// domain; a web console can be bolted on later without protocol changes.
package admin

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/qy-info/gosoul/internal/storage"
	"github.com/qy-info/gosoul/internal/user"
)

// Server serves the GM HTTP API.
type Server struct {
	log      *slog.Logger
	accounts *user.AccountService
	chars    *user.CharacterService
	wallets  *user.CurrencyService
}

// New builds the GM surface over the user services.
func New(log *slog.Logger, accounts *user.AccountService, chars *user.CharacterService, wallets *user.CurrencyService) *Server {
	return &Server{log: log, accounts: accounts, chars: chars, wallets: wallets}
}

// Handler returns the HTTP mux.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/admin/accounts", s.listAccounts)
	mux.HandleFunc("POST /api/admin/accounts", s.createAccount)
	mux.HandleFunc("GET /api/admin/accounts/{id}", s.getAccount)
	mux.HandleFunc("POST /api/admin/accounts/{id}/grant", s.grantAccount)
	return mux
}

func (s *Server) listAccounts(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	accounts, err := s.accounts.List(r.Context(), limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, accounts)
}

func (s *Server) createAccount(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Nickname string `json:"nickname"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "bad json")
		return
	}
	acc, err := s.accounts.Signup(r.Context(), body.Username, body.Password, body.Nickname)
	if errors.Is(err, user.ErrTaken) {
		writeErr(w, http.StatusConflict, "username taken")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Seed a fresh wallet.
	_ = s.wallets.NewAccountAllowance(r.Context(), acc.ID, 0, 0, 0)
	writeJSON(w, map[string]any{"id": acc.ID, "username": acc.Username, "nickname": acc.Nickname})
}

func (s *Server) getAccount(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	acc, err := s.accounts.Get(r.Context(), id)
	if errors.Is(err, storage.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "account not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, serializeAccount(r.Context(), acc, s.chars, s.wallets))
}

type grantBody struct {
	Gold       int64   `json:"gold"`
	Diamond    int64   `json:"diamond"`
	SkinTicket int64   `json:"skin_ticket"`
	Characters []int64 `json:"characters"`
}

func (s *Server) grantAccount(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var body grantBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "bad json")
		return
	}
	ctx := r.Context()
	if err := s.wallets.Grant(ctx, id, body.Gold, body.Diamond, body.SkinTicket); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, charID := range body.Characters {
		if err := s.chars.Grant(ctx, id, charID); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	acc, err := s.accounts.Get(ctx, id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "account not found")
		return
	}
	writeJSON(w, serializeAccount(ctx, acc, s.chars, s.wallets))
}

func serializeAccount(ctx context.Context, acc *storage.Account, chars *user.CharacterService, wallets *user.CurrencyService) map[string]any {
	cs, _ := chars.List(ctx, acc.ID)
	currency, _ := wallets.Balance(ctx, acc.ID)
	return map[string]any{
		"id":          acc.ID,
		"username":    acc.Username,
		"nickname":    acc.Nickname,
		"level_id":    acc.LevelID,
		"gold":        currency.Gold,
		"diamond":     currency.Diamond,
		"skin_ticket": currency.SkinTicket,
		"characters":  cs,
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	http.Error(w, msg, code)
}
