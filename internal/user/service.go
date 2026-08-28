// Package user implements the account-side domain: signup/login, character
// licenses and wallets. Services are pure Go (no protocol knowledge); the
// lobby/service and admin layers translate to and from RPC payloads.
package user

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/qy-info/gosoul/internal/storage"
)

// ErrTaken is returned when a signup collides with an existing username.
var ErrTaken = errors.New("user: username taken")

// AccountService owns registration and authentication.
type AccountService struct {
	accounts storage.AccountRepo
}

// NewAccountService wires the service to its repository.
func NewAccountService(accounts storage.AccountRepo) *AccountService {
	return &AccountService{accounts: accounts}
}

// Signup registers a new account. Passwords are bcrypt-hashed before storage.
func (s *AccountService) Signup(ctx context.Context, username, password, nickname string) (*storage.Account, error) {
	existing, err := s.accounts.GetByUsername(ctx, username)
	if err == nil && existing != nil {
		return nil, ErrTaken
	}
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}

	if nickname == "" {
		nickname = username
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	acc := &storage.Account{
		Username:     username,
		PasswordHash: string(hash),
		Nickname:     nickname,
		AvatarID:     400101,
		LevelID:      1001,
		CreatedAt:    time.Now().Unix(),
	}
	if err := s.accounts.Create(ctx, acc); err != nil {
		return nil, err
	}
	return acc, nil
}

// LoginOrAutoSignup is the private-server login path: known accounts verify
// their password; unknown usernames are auto-registered so the client never
// hits a registration wall.
func (s *AccountService) LoginOrAutoSignup(ctx context.Context, username, password string) (*storage.Account, error) {
	acc, err := s.accounts.GetByUsername(ctx, username)
	if errors.Is(err, storage.ErrNotFound) {
		return s.Signup(ctx, username, password, "")
	}
	if err != nil {
		return nil, err
	}
	if acc.PasswordHash != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(acc.PasswordHash), []byte(password)); err != nil {
			return nil, err
		}
	}
	_ = s.accounts.UpdateLogin(ctx, acc.ID, time.Now().Unix())
	return acc, nil
}

// Get returns an account by ID.
func (s *AccountService) Get(ctx context.Context, id int64) (*storage.Account, error) {
	return s.accounts.GetByID(ctx, id)
}

// CharacterService owns character licenses.
type CharacterService struct {
	chars storage.CharacterRepo
}

// NewCharacterService wires the service to its repository.
func NewCharacterService(chars storage.CharacterRepo) *CharacterService {
	return &CharacterService{chars: chars}
}

// List returns all characters held by an account.
func (s *CharacterService) List(ctx context.Context, accountID int64) ([]storage.Character, error) {
	return s.chars.List(ctx, accountID)
}

// Grant licenses a character to an account (idempotent).
func (s *CharacterService) Grant(ctx context.Context, accountID, charID int64) error {
	return s.chars.Add(ctx, storage.Character{AccountID: accountID, CharID: charID, Level: 1})
}

// CurrencyService owns wallets.
type CurrencyService struct {
	wallets storage.CurrencyRepo
}

// NewCurrencyService wires the service to its repository.
func NewCurrencyService(wallets storage.CurrencyRepo) *CurrencyService {
	return &CurrencyService{wallets: wallets}
}

// Balance returns the wallet of an account.
func (s *CurrencyService) Balance(ctx context.Context, accountID int64) (storage.Currency, error) {
	return s.wallets.Get(ctx, accountID)
}

// NewAccountAllowance seeds a fresh account with starting money.
func (s *CurrencyService) NewAccountAllowance(ctx context.Context, accountID int64, gold, diamond, tickets int64) error {
	if gold > 0 {
		if err := s.wallets.AddGold(ctx, accountID, gold); err != nil {
			return err
		}
	}
	if diamond > 0 {
		if err := s.wallets.AddDiamond(ctx, accountID, diamond); err != nil {
			return err
		}
	}
	return s.wallets.AddSkinTicket(ctx, accountID, tickets)
}

// Grant applies a mixed wallet grant to one account.
func (s *CurrencyService) Grant(ctx context.Context, accountID, gold, diamond, tickets int64) error {
	return s.NewAccountAllowance(ctx, accountID, gold, diamond, tickets)
}

// List returns accounts page-wise (used by the GM surface).
func (s *AccountService) List(ctx context.Context, limit int) ([]storage.Account, error) {
	return s.accounts.List(ctx, limit, 0)
}

// Touch records the last login timestamp.
func (s *AccountService) Touch(ctx context.Context, id int64, at int64) error {
	return s.accounts.UpdateLogin(ctx, id, at)
}
