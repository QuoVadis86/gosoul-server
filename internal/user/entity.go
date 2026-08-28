package user

import "errors"

// ErrNotFound is the domain-level miss error returned by repositories and
// services. It lives in the domain so storage implementations can return it
// without importing back into the domain.
var ErrNotFound = errors.New("user: not found")

// Account is a player record (domain entity).
type Account struct {
	ID           int64
	Username     string
	PasswordHash string
	Nickname     string
	AvatarID     int64
	LevelID      int64
	LevelScore   int64
	VIP          int64
	Title        int64
	Signature    string
	Verified     int64
	CreatedAt    int64
	LastLogin    int64
}

// Character is a licensed character on an account.
type Character struct {
	AccountID int64
	CharID    int64
	Level     int64
	Exp       int64
	SkinID    int64
}

// Wallet is an account's currency balance.
type Wallet struct {
	Gold       int64
	Diamond    int64
	SkinTicket int64
}

// Achievement tracks one achievement's runtime progress for an account.
type Achievement struct {
	AccountID int64
	AchieveID int64
	Progress  int64
	Rewarded  int64
}
