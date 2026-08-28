package shop

import (
	"context"
	"testing"
)

type fakeTeller struct {
	gold, diamond, tickets int64
}

func (f *fakeTeller) Balance(_ context.Context, _ int64) (int64, int64, int64, error) {
	return f.gold, f.diamond, f.tickets, nil
}
func (f *fakeTeller) Charge(_ context.Context, _, g, d, t int64) error {
	if f.gold+g < 0 || f.diamond+d < 0 || f.tickets+t < 0 {
		return ErrNotEnough
	}
	f.gold, f.diamond, f.tickets = f.gold+g, f.diamond+d, f.tickets+t
	return nil
}
func (f *fakeTeller) Grant(_ context.Context, _, g, d, t int64) error {
	f.gold, f.diamond, f.tickets = f.gold+g, f.diamond+d, f.tickets+t
	return nil
}

func TestPurchaseGold(t *testing.T) {
	teller := &fakeTeller{gold: 1000, diamond: 500}
	svc := New(teller, []Goods{{ID: 1, Cost: 100, Currency: Diamond, RewardID: 100, RewardCount: 5000}})
	slots, err := svc.Purchase(context.Background(), 1, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 1 || slots[0].ID != 100001 || slots[0].Count != 10000 {
		t.Fatalf("rewards = %+v", slots)
	}
	if teller.diamond != 300 || teller.gold != 11000 {
		t.Fatalf("balances = %+v", teller)
	}
}

func TestPurchaseUnknown(t *testing.T) {
	svc := New(&fakeTeller{gold: 100}, []Goods{{ID: 9, Cost: 1, Currency: Gold}})
	if _, err := svc.Purchase(context.Background(), 1, 404, 1); err != ErrNoGoods {
		t.Fatalf("err = %v, want ErrNoGoods", err)
	}
}

func TestPurchaseInsufficient(t *testing.T) {
	teller := &fakeTeller{gold: 10}
	svc := New(teller, []Goods{{ID: 2, Cost: 100, Currency: Gold}})
	if _, err := svc.Purchase(context.Background(), 1, 2, 1); err != ErrNotEnough {
		t.Fatalf("err = %v, want ErrNotEnough", err)
	}
	if teller.gold != 10 {
		t.Fatal("balance must not change on failure")
	}
}
