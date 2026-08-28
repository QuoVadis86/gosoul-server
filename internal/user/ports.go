package user

import "context"

// AccountRepo persists player records.
type AccountRepo interface {
	Create(ctx context.Context, a *Account) error
	GetByID(ctx context.Context, id int64) (*Account, error)
	GetByUsername(ctx context.Context, username string) (*Account, error)
	List(ctx context.Context, limit, offset int) ([]Account, error)
	UpdateLogin(ctx context.Context, id int64, lastLogin int64) error
}

// CharacterRepo persists character licenses.
type CharacterRepo interface {
	List(ctx context.Context, accountID int64) ([]Character, error)
	Add(ctx context.Context, c Character) error
}

// WalletRepo persists currency balances.
type WalletRepo interface {
	Get(ctx context.Context, accountID int64) (Wallet, error)
	AddGold(ctx context.Context, accountID, delta int64) error
	AddDiamond(ctx context.Context, accountID, delta int64) error
	AddSkinTicket(ctx context.Context, accountID, delta int64) error
}

// AchieveRepo persists achievement progress.
type AchieveRepo interface {
	List(ctx context.Context, accountID int64) ([]Achievement, error)
	Set(ctx context.Context, a Achievement) error
}
