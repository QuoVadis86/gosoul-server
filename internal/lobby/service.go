// Package lobby implements the account-facing lobby domain: login, home
// data, rooms and matching. It is protocol-agnostic — the transport layer
// maps liqi RPC payloads onto these calls.
package lobby

import (
	"context"
	"errors"
	"time"

	"github.com/qy-info/gosoul/internal/storage"
	"github.com/qy-info/gosoul/internal/user"
)

// DefaultAllowance seeds new accounts when they first log in.
var DefaultAllowance = struct{ Gold, Diamond, SkinTicket int64 }{100000, 1000, 100}

// LoginRequest mirrors the client's login payload fields.
type LoginRequest struct {
	Username string
	Password string
}

// AccountState is the full home-data projection the client renders after login.
type AccountState struct {
	Account    *storage.Account
	Gold       int64
	Diamond    int64
	SkinTicket int64
	Characters []storage.Character
}

// Service is the lobby domain service.
type Service struct {
	accounts *user.AccountService
	chars    *user.CharacterService
	wallets  *user.CurrencyService
}

// NewService wires the lobby over the user domain.
func NewService(accounts *user.AccountService, chars *user.CharacterService, wallets *user.CurrencyService) *Service {
	return &Service{accounts: accounts, chars: chars, wallets: wallets}
}

// ErrNotFound wraps the storage-level miss.
var ErrNotFound = storage.ErrNotFound

// Login authenticates (or auto-registers) and returns the home state.
func (s *Service) Login(ctx context.Context, req LoginRequest) (*AccountState, error) {
	acc, err := s.accounts.LoginOrAutoSignup(ctx, req.Username, req.Password)
	if err != nil {
		return nil, err
	}

	// First visit seeds the wallet.
	if _, err := s.wallets.Balance(ctx, acc.ID); err != nil && errors.Is(err, storage.ErrNotFound) {
		_ = s.wallets.NewAccountAllowance(ctx, acc.ID, DefaultAllowance.Gold, DefaultAllowance.Diamond, DefaultAllowance.SkinTicket)
	}

	return s.Home(ctx, acc.ID)
}

// Home assembles everything the lobby screen needs for one account.
func (s *Service) Home(ctx context.Context, accountID int64) (*AccountState, error) {
	acc, err := s.accounts.Get(ctx, accountID)
	if err != nil {
		return nil, err
	}
	currency, err := s.wallets.Balance(ctx, accountID)
	if err != nil {
		return nil, err
	}
	chars, err := s.chars.List(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return &AccountState{
		Account:    acc,
		Gold:       currency.Gold,
		Diamond:    currency.Diamond,
		SkinTicket: currency.SkinTicket,
		Characters: chars,
	}, nil
}

// Touch updates login bookkeeping; used by the RPC layer on heartbeat/login.
func (s *Service) Touch(ctx context.Context, accountID int64) error {
	return s.accounts.Touch(ctx, accountID, time.Now().Unix())
}
