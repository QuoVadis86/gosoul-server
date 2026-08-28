// Package gacha implements the character lottery (抽奖): paid draws from a
// character pool. The pool is deterministic (client-visible list), while the
// per-draw choice uses the provided RNG so tests can pin seeds.
package gacha

import (
	"context"
	"errors"
	"math/rand"
)

// ErrNotEnough is returned when a draw would exceed the account's diamonds.
var ErrNotEnough = errors.New("gacha: not enough diamonds")

// PricePerDraw is the diamond cost of a single draw.
const PricePerDraw = 100

// Drawer is the account authority the gacha consults for payment and rewards.
type Drawer interface {
	// Balance returns the account's wallet.
	Balance(ctx context.Context, accountID int64) (Diamond int64, err error)
	// Charge deducts diamonds; must fail when the balance is insufficient.
	Charge(ctx context.Context, accountID, delta int64) error
	// GrantCharacter licenses one character (idempotent).
	GrantCharacter(ctx context.Context, accountID, charID int64) error
}

// Pool is the list of characters a gacha can award. index is the client-facing
// character id.
type Pool struct {
	Characters []int64
}

// Service runs draws for a single pool.
type Service struct {
	draw Drawer
	pool Pool
	rng  *rand.Rand
}

// New wires the service over its authorities.
func New(d Drawer, pool Pool, rng *rand.Rand) *Service {
	return &Service{draw: d, pool: pool, rng: rng}
}

// Result is one gacha outcome.
type Result struct {
	// ResultList is the drawn character ids, one per paid draw.
	ResultList []int64
	// SpRewardItems mirrors optional bonus grant (unused for now).
	SpRewardItems []int64
	// RemainCount mirrors how many draws remain (unused; always 0).
	RemainCount uint32
}

// Open performs count paid draws for an account. A successful call always
// returns exactly count results.
func (s *Service) Open(ctx context.Context, accountID int64, count uint32) (*Result, error) {
	if count == 0 {
		count = 1
	}
	cost := int64(count) * PricePerDraw
	diamond, err := s.draw.Balance(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if diamond < cost {
		return nil, ErrNotEnough
	}
	if err := s.draw.Charge(ctx, accountID, -cost); err != nil {
		return nil, err
	}
	res := &Result{}
	for i := uint32(0); i < count; i++ {
		var id int64
		if len(s.pool.Characters) > 0 {
			id = s.pool.Characters[s.rng.Intn(len(s.pool.Characters))]
		} else {
			id = 200001
		}
		if err := s.draw.GrantCharacter(ctx, accountID, id); err != nil {
			return nil, err
		}
		res.ResultList = append(res.ResultList, id)
	}
	return res, nil
}
