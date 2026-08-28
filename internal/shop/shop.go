// Package shop implements a small static goods catalog purchasable with
// currency. Goods entries live in code for now; a DB-driven catalog can replace
// the catalog function without touching the purchase path.
package shop

import (
	"context"
	"errors"
)

// ErrNoGoods is returned when the goods id is unknown.
var ErrNoGoods = errors.New("shop: unknown goods")

// ErrNotEnough is returned when the account cannot afford an item.
var ErrNotEnough = errors.New("shop: not enough currency")

// Currency enumerates the spendable coin types.
type Currency int

const (
	Gold Currency = iota
	Diamond
	SkinTicket
)

// Goods is one purchasable entry.
type Goods struct {
	ID          uint32
	Cost        int64
	Currency    Currency
	RewardID    uint32 // numeric id of the granted thing (e.g. item or character)
	RewardCount int64
}

// Teller is the account authority for payments and grants.
type Teller interface {
	// Balance returns current spendable amounts (gold, diamond, tickets).
	Balance(ctx context.Context, accountID int64) (gold, diamond, tickets int64, err error)
	// Charge applies a signed currency delta (negative to spend).
	Charge(ctx context.Context, accountID, gold, diamond, tickets int64) error
	// Grant applies a positive currency delta (rewards).
	Grant(ctx context.Context, accountID, gold, diamond, tickets int64) error
}

// Service sells catalog entries.
type Service struct {
	teller  Teller
	catalog map[uint32]Goods
}

// New wires the service over its teller and catalog.
func New(t Teller, catalog []Goods) *Service {
	m := make(map[uint32]Goods, len(catalog))
	for _, g := range catalog {
		m[g.ID] = g
	}
	return &Service{teller: t, catalog: m}
}

// Slot is one rewarded currency/item amount.
type Slot struct {
	ID    uint32
	Count uint32
}

// Purchase buys count copies, charging the account and returning the reward
// summary (id,count pairs).
func (s *Service) Purchase(ctx context.Context, accountID int64, goodsID uint32, count int64) ([]Slot, error) {
	g, ok := s.catalog[goodsID]
	if !ok {
		return nil, ErrNoGoods
	}
	if count <= 0 {
		count = 1
	}
	total := g.Cost * count
	barG, barD, barT, err := s.teller.Balance(ctx, accountID)
	if err != nil {
		return nil, err
	}
	switch g.Currency {
	case Diamond:
		if barD < total {
			return nil, ErrNotEnough
		}
	case SkinTicket:
		if barT < total {
			return nil, ErrNotEnough
		}
	default:
		if barG < total {
			return nil, ErrNotEnough
		}
	}
	if err := s.charge(ctx, accountID, g.Currency, -total); err != nil {
		return nil, err
	}
	switch {
	case g.RewardID == 100: // gold bucket
		return []Slot{{ID: 100001, Count: uint32(g.RewardCount * count)}}, s.teller.Grant(ctx, accountID, g.RewardCount*count, 0, 0)
	case g.RewardID == 101: // diamond bucket
		return []Slot{{ID: 100002, Count: uint32(g.RewardCount * count)}}, s.teller.Grant(ctx, accountID, 0, g.RewardCount*count, 0)
	default:
		return []Slot{{ID: g.RewardID, Count: uint32(g.RewardCount * count)}}, nil
	}
}

func (s *Service) charge(ctx context.Context, accountID int64, cur Currency, total int64) error {
	switch cur {
	case Diamond:
		return s.teller.Charge(ctx, accountID, 0, total, 0)
	case SkinTicket:
		return s.teller.Charge(ctx, accountID, 0, 0, total)
	default:
		return s.teller.Charge(ctx, accountID, total, 0, 0)
	}
}
