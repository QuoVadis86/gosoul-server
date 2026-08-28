package storage

import (
	"context"
	"database/sql"
	"errors"
)

type accountRepo struct{ db *sql.DB }

func (r *accountRepo) Create(ctx context.Context, a *Account) error {
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

	// Every account starts with a wallet row.
	_, err = r.db.ExecContext(ctx, `INSERT INTO currencies(account_id) VALUES(?)`, id)
	return err
}

func (r *accountRepo) GetByID(ctx context.Context, id int64) (*Account, error) {
	return scanAccount(r.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, nickname, avatar_id, level_id, level_score, vip, title, signature, created_at, last_login
		 FROM accounts WHERE id = ?`, id))
}

func (r *accountRepo) List(ctx context.Context, limit, offset int) ([]Account, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, username, password_hash, nickname, avatar_id, level_id, level_score, vip, title, signature, created_at, last_login
		 FROM accounts ORDER BY id LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.Username, &a.PasswordHash, &a.Nickname, &a.AvatarID,
			&a.LevelID, &a.LevelScore, &a.VIP, &a.Title, &a.Signature, &a.CreatedAt, &a.LastLogin); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *accountRepo) GetByUsername(ctx context.Context, username string) (*Account, error) {
	return scanAccount(r.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, nickname, avatar_id, level_id, level_score, vip, title, signature, created_at, last_login
		 FROM accounts WHERE username = ?`, username))
}

func (r *accountRepo) UpdateLogin(ctx context.Context, id int64, lastLogin int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE accounts SET last_login = ? WHERE id = ?`, lastLogin, id)
	return err
}

func scanAccount(row *sql.Row) (*Account, error) {
	var a Account
	err := row.Scan(&a.ID, &a.Username, &a.PasswordHash, &a.Nickname, &a.AvatarID,
		&a.LevelID, &a.LevelScore, &a.VIP, &a.Title, &a.Signature, &a.CreatedAt, &a.LastLogin)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

type characterRepo struct{ db *sql.DB }

func (r *characterRepo) List(ctx context.Context, accountID int64) ([]Character, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT account_id, charid, level, exp, skin_id FROM characters WHERE account_id = ?`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Character
	for rows.Next() {
		var c Character
		if err := rows.Scan(&c.AccountID, &c.CharID, &c.Level, &c.Exp, &c.SkinID); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *characterRepo) Add(ctx context.Context, c Character) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO characters(account_id, charid, level, exp, skin_id) VALUES(?,?,?,?,?)`,
		c.AccountID, c.CharID, c.Level, c.Exp, c.SkinID)
	return err
}

type currencyRepo struct{ db *sql.DB }

func (r *currencyRepo) Get(ctx context.Context, accountID int64) (Currency, error) {
	var c Currency
	err := r.db.QueryRowContext(ctx,
		`SELECT gold, diamond, skin_ticket FROM currencies WHERE account_id = ?`, accountID).
		Scan(&c.Gold, &c.Diamond, &c.SkinTicket)
	if errors.Is(err, sql.ErrNoRows) {
		return c, ErrNotFound
	}
	return c, err
}

func (r *currencyRepo) add(ctx context.Context, accountID, delta int64, column string) error {
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
		return ErrNotFound
	}
	return nil
}

func (r *currencyRepo) AddGold(ctx context.Context, accountID, delta int64) error {
	return r.add(ctx, accountID, delta, "gold")
}

func (r *currencyRepo) AddDiamond(ctx context.Context, accountID, delta int64) error {
	return r.add(ctx, accountID, delta, "diamond")
}

func (r *currencyRepo) AddSkinTicket(ctx context.Context, accountID, delta int64) error {
	return r.add(ctx, accountID, delta, "skin_ticket")
}
