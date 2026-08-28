// Package storage implements the persistence ports defined by the domain
// (internal/user). It owns migrations and SQLite-backed repositories only;
// entities and error contracts live in the domain, and storage depends on the
// domain rather than the other way around.
package storage

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	_ "modernc.org/sqlite"

	"github.com/qy-info/gosoul/internal/paipu"
	"github.com/qy-info/gosoul/internal/user"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Store bundles the repository implementations behind one object.
type Store struct {
	db *sql.DB

	Account   user.AccountRepo
	Character user.CharacterRepo
	Wallet    user.WalletRepo
	Achieve   user.AchieveRepo
	Paipu     paipu.Store
}

// Open boots SQLite at path and applies all migrations.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("storage: open %s: %w", path, err)
	}
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
	return &Store{
		db:        db,
		Account:   &accountRepo{db: db},
		Character: &characterRepo{db: db},
		Wallet:    &walletRepo{db: db},
		Achieve:   &achieveRepo{db: db},
		Paipu:     &paipuRepo{db: db},
	}, nil
}

// Close releases the underlying database.
func (s *Store) Close() error { return s.db.Close() }

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
	for _, c := range name {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int64(c-'0')
	}
	return n
}
