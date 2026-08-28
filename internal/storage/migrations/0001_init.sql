CREATE TABLE IF NOT EXISTS accounts (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  username      TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL DEFAULT '',
  nickname      TEXT NOT NULL DEFAULT '',
  avatar_id     INTEGER NOT NULL DEFAULT 400101,
  level_id      INTEGER NOT NULL DEFAULT 1001,
  level_score   INTEGER NOT NULL DEFAULT 0,
  vip           INTEGER NOT NULL DEFAULT 0,
  title         INTEGER NOT NULL DEFAULT 0,
  signature     TEXT NOT NULL DEFAULT '',
  verified      INTEGER NOT NULL DEFAULT 0,
  created_at    INTEGER NOT NULL,
  last_login    INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS currencies (
  account_id   INTEGER PRIMARY KEY REFERENCES accounts(id),
  gold         INTEGER NOT NULL DEFAULT 0,
  diamond      INTEGER NOT NULL DEFAULT 0,
  skin_ticket  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS characters (
  account_id INTEGER NOT NULL REFERENCES accounts(id),
  charid     INTEGER NOT NULL,
  level      INTEGER NOT NULL DEFAULT 1,
  exp        INTEGER NOT NULL DEFAULT 0,
  skin_id    INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (account_id, charid)
);

CREATE TABLE IF NOT EXISTS inventory (
  account_id INTEGER NOT NULL REFERENCES accounts(id),
  item_id    INTEGER NOT NULL,
  stack      INTEGER NOT NULL DEFAULT 1,
  PRIMARY KEY (account_id, item_id)
);

CREATE TABLE IF NOT EXISTS achievements (
  account_id INTEGER NOT NULL REFERENCES accounts(id),
  ach_id     INTEGER NOT NULL,
  progress   INTEGER NOT NULL DEFAULT 0,
  rewarded   INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (account_id, ach_id)
);

CREATE TABLE IF NOT EXISTS paipu (
  uuid       TEXT PRIMARY KEY,
  json       TEXT NOT NULL,
  created_at INTEGER NOT NULL
);
