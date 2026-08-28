package storage

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/qy-info/gosoul/internal/user"
)

func openTest(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestAccountCRUD(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()

	acc := &user.Account{Username: "alice", PasswordHash: "h", Nickname: "Alice", CreatedAt: 1}
	if err := s.Account.Create(ctx, acc); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if acc.ID == 0 {
		t.Fatal("Create did not assign ID")
	}

	got, err := s.Account.GetByUsername(ctx, "alice")
	if err != nil {
		t.Fatalf("GetByUsername: %v", err)
	}
	if got.ID != acc.ID || got.Nickname != "Alice" {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}

	if _, err := s.Account.GetByUsername(ctx, "nobody"); !errors.Is(err, user.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestWalletSeededOnCreate(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	acc := &user.Account{Username: "bob", CreatedAt: 1}
	if err := s.Account.Create(ctx, acc); err != nil {
		t.Fatal(err)
	}
	c, err := s.Wallet.Get(ctx, acc.ID)
	if err != nil {
		t.Fatalf("wallet missing: %v", err)
	}
	if c.Gold != 0 || c.Diamond != 0 {
		t.Fatalf("seed mismatch: %+v", c)
	}
	if err := s.Wallet.AddGold(ctx, acc.ID, 500); err != nil {
		t.Fatal(err)
	}
	c, _ = s.Wallet.Get(ctx, acc.ID)
	if c.Gold != 500 {
		t.Fatalf("gold = %d", c.Gold)
	}
}

func TestCharacterGrantIdempotent(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	acc := &user.Account{Username: "carol", CreatedAt: 1}
	if err := s.Account.Create(ctx, acc); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := s.Character.Add(ctx, user.Character{AccountID: acc.ID, CharID: 200001, Level: 1}); err != nil {
			t.Fatalf("Add #%d: %v", i, err)
		}
	}
	list, err := s.Character.List(ctx, acc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("duplicate license stored: %d rows", len(list))
	}
}

func TestMigrationIdempotent(t *testing.T) {
	s := openTest(t)
	s.Close()
	s2, err := Open(filepath.Join(t.TempDir(), "x.db"))
	if err != nil {
		t.Fatalf("reopen fresh db: %v", err)
	}
	defer s2.Close()
}

func TestAchievementSetAndList(t *testing.T) {
	db, err := Open(t.TempDir() + "/ach.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	acc := &user.Account{Username: "achu", PasswordHash: "h", Nickname: "A", CreatedAt: 1}
	if err := db.Account.Create(context.Background(), acc); err != nil {
		t.Fatal(err)
	}
	entry := user.Achievement{AccountID: acc.ID, AchieveID: 1001, Progress: 3, Rewarded: 1}
	if err := db.Achieve.Set(context.Background(), entry); err != nil {
		t.Fatal(err)
	}
	got, err := db.Achieve.List(context.Background(), acc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].AchieveID != 1001 || got[0].Progress != 3 {
		t.Fatalf("achievements = %+v", got)
	}
	// upsert overwrites progress
	entry.Progress = 5
	if err := db.Achieve.Set(context.Background(), entry); err != nil {
		t.Fatal(err)
	}
	got, _ = db.Achieve.List(context.Background(), acc.ID)
	if got[0].Progress != 5 {
		t.Fatalf("progress after upsert = %d, want 5", got[0].Progress)
	}
}
