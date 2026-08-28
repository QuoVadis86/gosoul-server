// Package user implements the account-side domain: signup/login, wallets and
// character licenses, plus the composition read model the lobby surfaces.
// It is protocol-agnostic: protocol DTOs and handlers live in the surface
// packages (lobby for the client RPC face, admin for the GM face).
package user

import (
	"context"
	"crypto/subtle"
	"errors"
	"time"
)

// ErrTaken is returned when a signup collides with an existing username.
var ErrTaken = errors.New("user: username taken")

// ErrBadPassword is returned when a login supplies the wrong password.
var ErrBadPassword = errors.New("user: bad password")

// DefaultAllowance seeds new accounts on their first login.
var DefaultAllowance = struct{ Gold, Diamond, SkinTicket int64 }{100000, 1000, 100}

// Service is the aggregated account domain. It composes the four portraits
// (accounts, characters, wallets, achievements) so read models like Home are
// assembled with one call from the domain side.
type Service struct {
	accounts AccountRepo
	chars    CharacterRepo
	wallets  WalletRepo
	achv     AchieveRepo
}

// NewService wires the domain over its persistence ports.
func NewService(a AccountRepo, c CharacterRepo, w WalletRepo, ach AchieveRepo) *Service {
	return &Service{accounts: a, chars: c, wallets: w, achv: ach}
}

// Achievements lists every recorded achievement entry for an account.
func (s *Service) Achievements(ctx context.Context, accountID int64) ([]Achievement, error) {
	return s.achv.List(ctx, accountID)
}

// SetAchievement records or updates one achievement's progress.
func (s *Service) SetAchievement(ctx context.Context, a Achievement) error {
	return s.achv.Set(ctx, a)
}

// Signup registers a new account. The client already digests the password
// (Majsoul digests it before transmission), so the transmitted credential is
// stored verbatim and compared verbatim at login; hashing happens client-side.
func (s *Service) Signup(ctx context.Context, username, password, nickname string) (*Account, error) {
	existing, err := s.accounts.GetByUsername(ctx, username)
	if err == nil && existing != nil {
		return nil, ErrTaken
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if nickname == "" {
		nickname = username
	}
	acc := &Account{
		Username:     username,
		PasswordHash: password,
		Nickname:     nickname,
		AvatarID:     400101,
		LevelID:      1001,
		CreatedAt:    time.Now().Unix(),
	}
	if err := s.accounts.Create(ctx, acc); err != nil {
		return nil, err
	}
	// Registration carries an initial wallet.
	_ = s.wallets.AddGold(ctx, acc.ID, DefaultAllowance.Gold)
	_ = s.wallets.AddDiamond(ctx, acc.ID, DefaultAllowance.Diamond)
	_ = s.wallets.AddSkinTicket(ctx, acc.ID, DefaultAllowance.SkinTicket)
	return acc, nil
}

// Login authenticates a known account. Unknown usernames and wrong passwords
// are real errors — registration is an explicit signup, matching the official
// client flow.
func (s *Service) Login(ctx context.Context, username, password string) (*Account, error) {
	acc, err := s.accounts.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if acc.PasswordHash != "" {
		if subtle.ConstantTimeCompare([]byte(acc.PasswordHash), []byte(password)) != 1 {
			return nil, ErrBadPassword
		}
	}
	_ = s.accounts.UpdateLogin(ctx, acc.ID, time.Now().Unix())
	return acc, nil
}

// SignupByToken creates (or reuses) an account keyed by a visitor token. The
// token is stored as the username, mirroring how the official client treats
// locally-generated visitor identifiers as real accounts.
func (s *Service) SignupByToken(ctx context.Context, tokenKey string) (*Account, error) {
	existing, err := s.accounts.GetByUsername(ctx, tokenKey)
	if err == nil && existing != nil {
		return existing, nil
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	return s.Signup(ctx, tokenKey, "", "visitor")
}

// LoginByToken authenticates a visitor token (account keyed by that token).
func (s *Service) LoginByToken(ctx context.Context, tokenKey string) (*Account, error) {
	acc, err := s.accounts.GetByUsername(ctx, tokenKey)
	if err != nil {
		return nil, err
	}
	_ = s.accounts.UpdateLogin(ctx, acc.ID, time.Now().Unix())
	return acc, nil
}

// Get returns an account by ID.
func (s *Service) Get(ctx context.Context, id int64) (*Account, error) {
	return s.accounts.GetByID(ctx, id)
}

// List returns accounts page-wise (used by the GM surface).
func (s *Service) List(ctx context.Context, limit int) ([]Account, error) {
	return s.accounts.List(ctx, limit, 0)
}

// Touch records the last login timestamp.
func (s *Service) Touch(ctx context.Context, id int64, at int64) error {
	return s.accounts.UpdateLogin(ctx, id, at)
}

// Balance returns the wallet of an account.
func (s *Service) Balance(ctx context.Context, accountID int64) (Wallet, error) {
	return s.wallets.Get(ctx, accountID)
}

// Grant applies a mixed wallet change to one account.
func (s *Service) Grant(ctx context.Context, accountID, gold, diamond, tickets int64) error {
	if gold != 0 {
		if err := s.wallets.AddGold(ctx, accountID, gold); err != nil {
			return err
		}
	}
	if diamond != 0 {
		if err := s.wallets.AddDiamond(ctx, accountID, diamond); err != nil {
			return err
		}
	}
	return s.wallets.AddSkinTicket(ctx, accountID, tickets)
}

// Characters lists everything an account owns.
func (s *Service) Characters(ctx context.Context, accountID int64) ([]Character, error) {
	return s.chars.List(ctx, accountID)
}

// GrantCharacter licenses a character (idempotent).
func (s *Service) GrantCharacter(ctx context.Context, accountID, charID int64) error {
	return s.chars.Add(ctx, Character{AccountID: accountID, CharID: charID, Level: 1})
}

// Home is the composition read model the lobby and GM surfaces render.
type Home struct {
	Account    *Account
	Wallet     Wallet
	Characters []Character
}

// Home assembles everything a seat needs after login or when entering lobby.
func (s *Service) Home(ctx context.Context, accountID int64) (*Home, error) {
	acc, err := s.accounts.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	wallet, err := s.wallets.Get(ctx, accountID)
	if errors.Is(err, ErrNotFound) {
		wallet = Wallet{}
	} else if err != nil {
		return nil, err
	}
	chars, err := s.chars.List(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return &Home{Account: acc, Wallet: wallet, Characters: chars}, nil
}

// GetByUsernameToken exposes existence lookup by the token-keyed username.
func (s *Service) GetByUsernameToken(ctx context.Context, tokenKey string) (*Account, error) {
	return s.accounts.GetByUsername(ctx, tokenKey)
}
