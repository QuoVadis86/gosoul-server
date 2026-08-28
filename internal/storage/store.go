// Package storage owns persistence. Every repository is defined by an
// interface in this package; the SQLite implementation lives alongside it and
// a Postgres adapter can be swapped in without touching callers (see
// ADR-0001 D2).
package storage

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// ErrNotFound is returned when a queried row does not exist.
var ErrNotFound = errors.New("storage: not found")

// Account is a user record.
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

// AccountRepo persists user records.
type AccountRepo interface {
	Create(ctx context.Context, a *Account) error
	GetByID(ctx context.Context, id int64) (*Account, error)
	List(ctx context.Context, limit, offset int) ([]Account, error)
	GetByUsername(ctx context.Context, username string) (*Account, error)
	UpdateLogin(ctx context.Context, id int64, lastLogin int64) error
}

// Character is a licensed character on an account.
type Character struct {
	AccountID int64
	CharID    int64
	Level     int64
	Exp       int64
	SkinID    int64
}

// CharacterRepo persists character licenses.
type CharacterRepo interface {
	List(ctx context.Context, accountID int64) ([]Character, error)
	Add(ctx context.Context, c Character) error
}

// Currency is an account's wallet.
type Currency struct {
	Gold       int64
	Diamond    int64
	SkinTicket int64
}

// CurrencyRepo persists wallets.
type CurrencyRepo interface {
	Get(ctx context.Context, accountID int64) (Currency, error)
	AddGold(ctx context.Context, accountID, delta int64) error
	AddDiamond(ctx context.Context, accountID, delta int64) error
	AddSkinTicket(ctx context.Context, accountID, delta int64) error
}

// Store bundles the repositories behind one object.
type Store struct {
	db *sql.DB

	Account   AccountRepo
	Character CharacterRepo
	Currency  CurrencyRepo
}

// Open boots SQLite at path and applies all migrations.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("storage: open %s: %w", path, err)
	}
	// SQLite with a single writer per process is the norm for this project.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("storage: journal_mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("storage: foreign_keys: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return newStore(db), nil
}

// Close releases the underlying database.
func (s *Store) Close() error { return s.db.Close() }

func newStore(db *sql.DB) *Store {
	return &Store{
		db:        db,
		Account:   &accountRepo{db: db},
		Character: &characterRepo{db: db},
		Currency:  &currencyRepo{db: db},
	}
}

// migrate applies embedded SQL files in lexical order within a single write
// transaction guarded by a user version marker.
func migrate(db *sql.DB) error {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	current, err := userVersion(db)
	if err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		version := versionOf(name)
		if version == 0 || version <= current {
			continue
		}
		data, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(data)); err != nil {
			tx.Rollback()
			return fmt.Errorf("storage: migrate %s: %w", name, err)
		}
		if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", version)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func userVersion(db *sql.DB) (int64, error) {
	var v int64
	err := db.QueryRow("PRAGMA user_version").Scan(&v)
	return v, err
}

func versionOf(name string) int64 {
	var n int64
	consumed := 0
	for _, c := range name {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int64(c-'0')
		consumed++
	}
	if consumed == 0 {
		return 0
	}
	return n
}
