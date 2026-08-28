package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/qy-info/gosoul/internal/paipu"
	"github.com/qy-info/gosoul/internal/user"
)

type accountRepo struct{ db *sql.DB }

func (r *accountRepo) Create(ctx context.Context, a *user.Account) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO accounts(username, password_hash, nickname, avatar_id, level_id, level_score, vip, title, signature, created_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?)`,
		a.Username, a.PasswordHash, a.Nickname, a.AvatarID, a.LevelID, a.LevelScore, a.VIP, a.Title, a.Signature, a.CreatedAt)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	a.ID = id

	_, err = r.db.ExecContext(ctx, `INSERT INTO currencies(account_id) VALUES(?)`, id)
	return err
}

func (r *accountRepo) GetByID(ctx context.Context, id int64) (*user.Account, error) {
	return scanAccount(r.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, nickname, avatar_id, level_id, level_score, vip, title, signature, verified, created_at, last_login
		 FROM accounts WHERE id = ?`, id))
}

func (r *accountRepo) GetByUsername(ctx context.Context, username string) (*user.Account, error) {
	return scanAccount(r.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, nickname, avatar_id, level_id, level_score, vip, title, signature, verified, created_at, last_login
		 FROM accounts WHERE username = ?`, username))
}

func (r *accountRepo) List(ctx context.Context, limit, offset int) ([]user.Account, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, username, password_hash, nickname, avatar_id, level_id, level_score, vip, title, signature, verified, created_at, last_login
		 FROM accounts ORDER BY id LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []user.Account
	for rows.Next() {
		var a user.Account
		if err := rows.Scan(&a.ID, &a.Username, &a.PasswordHash, &a.Nickname, &a.AvatarID,
			&a.LevelID, &a.LevelScore, &a.VIP, &a.Title, &a.Signature, &a.Verified, &a.CreatedAt, &a.LastLogin); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *accountRepo) UpdateLogin(ctx context.Context, id int64, lastLogin int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE accounts SET last_login = ? WHERE id = ?`, lastLogin, id)
	return err
}

func scanAccount(row *sql.Row) (*user.Account, error) {
	var a user.Account
	err := row.Scan(&a.ID, &a.Username, &a.PasswordHash, &a.Nickname, &a.AvatarID,
		&a.LevelID, &a.LevelScore, &a.VIP, &a.Title, &a.Signature, &a.Verified, &a.CreatedAt, &a.LastLogin)
	if err == sql.ErrNoRows {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

type characterRepo struct{ db *sql.DB }

func (r *characterRepo) List(ctx context.Context, accountID int64) ([]user.Character, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT account_id, charid, level, exp, skin_id FROM characters WHERE account_id = ?`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []user.Character
	for rows.Next() {
		var c user.Character
		if err := rows.Scan(&c.AccountID, &c.CharID, &c.Level, &c.Exp, &c.SkinID); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *characterRepo) Add(ctx context.Context, c user.Character) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO characters(account_id, charid, level, exp, skin_id) VALUES(?,?,?,?,?)`,
		c.AccountID, c.CharID, c.Level, c.Exp, c.SkinID)
	return err
}

type walletRepo struct{ db *sql.DB }

func (r *walletRepo) Get(ctx context.Context, accountID int64) (user.Wallet, error) {
	var c user.Wallet
	err := r.db.QueryRowContext(ctx,
		`SELECT gold, diamond, skin_ticket FROM currencies WHERE account_id = ?`, accountID).
		Scan(&c.Gold, &c.Diamond, &c.SkinTicket)
	if err == sql.ErrNoRows {
		return c, user.ErrNotFound
	}
	return c, err
}

func (r *walletRepo) add(ctx context.Context, accountID, delta int64, column string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE currencies SET `+column+` = `+column+` + ? WHERE account_id = ?`, delta, accountID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return user.ErrNotFound
	}
	return nil
}

func (r *walletRepo) AddGold(ctx context.Context, accountID, delta int64) error {
	return r.add(ctx, accountID, delta, "gold")
}

func (r *walletRepo) AddDiamond(ctx context.Context, accountID, delta int64) error {
	return r.add(ctx, accountID, delta, "diamond")
}

func (r *walletRepo) AddSkinTicket(ctx context.Context, accountID, delta int64) error {
	return r.add(ctx, accountID, delta, "skin_ticket")
}

// paipuRepo persists finished game records.
type paipuRepo struct{ db *sql.DB }

func (r *paipuRepo) Save(ctx context.Context, rec paipu.Record) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO paipu(uuid, json, created_at) VALUES(?, ?, ?)
		 ON CONFLICT(uuid) DO UPDATE SET json=excluded.json, created_at=excluded.created_at`,
		rec.UUID, rec.JSON, rec.CreatedAt.Unix())
	return err
}

func (r *paipuRepo) Get(ctx context.Context, uuid string) (*paipu.Record, error) {
	row := r.db.QueryRowContext(ctx, `SELECT uuid, json, created_at FROM paipu WHERE uuid=?`, uuid)
	var rec paipu.Record
	var ts int64
	if err := row.Scan(&rec.UUID, &rec.JSON, &ts); err != nil {
		return nil, err
	}
	rec.CreatedAt = time.Unix(ts, 0)
	return &rec, nil
}

func (r *paipuRepo) List(ctx context.Context, limit int) ([]paipu.Record, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT uuid, json, created_at FROM paipu ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []paipu.Record
	for rows.Next() {
		var rec paipu.Record
		var ts int64
		if err := rows.Scan(&rec.UUID, &rec.JSON, &ts); err != nil {
			return nil, err
		}
		rec.CreatedAt = time.Unix(ts, 0)
		out = append(out, rec)
	}
	return out, rows.Err()
}

// achieveRepo persists achievement progress rows.
type achieveRepo struct{ db *sql.DB }

func (r *achieveRepo) List(ctx context.Context, accountID int64) ([]user.Achievement, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT account_id, ach_id, progress, rewarded FROM achievements WHERE account_id=?`,
		accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []user.Achievement
	for rows.Next() {
		var a user.Achievement
		if err := rows.Scan(&a.AccountID, &a.AchieveID, &a.Progress, &a.Rewarded); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *achieveRepo) Set(ctx context.Context, a user.Achievement) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO achievements(account_id, ach_id, progress, rewarded)
		 VALUES(?, ?, ?, ?)
		 ON CONFLICT(account_id, ach_id) DO UPDATE SET progress=excluded.progress, rewarded=excluded.rewarded`,
		a.AccountID, a.AchieveID, a.Progress, a.Rewarded)
	return err
}
