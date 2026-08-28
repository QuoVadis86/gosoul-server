// Package paipu stores and queries finished game records (牌谱). A record is
// an opaque JSON document keyed by a UUID; the session layer serializes the
// final game state into this shape when a match ends.
package paipu

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when a uuid has no record.
var ErrNotFound = errors.New("paipu: not found")

// Record is one finished game's full replay payload.
type Record struct {
	UUID      string
	JSON      string
	CreatedAt time.Time
}

// Store is the persistence seam for paipu records.
type Store interface {
	// Save upserts a record by uuid.
	Save(ctx context.Context, r Record) error
	// Get returns a single record.
	Get(ctx context.Context, uuid string) (*Record, error)
	// List returns the most recent records, newest first.
	List(ctx context.Context, limit int) ([]Record, error)
}

// Service is the paipu domain over the store.
type Service struct {
	store Store
}

// New wires the service over its store.
func New(s Store) *Service {
	return &Service{store: s}
}

// Save records a finished game.
func (s *Service) Save(ctx context.Context, uuid, payload string) error {
	return s.store.Save(ctx, Record{UUID: uuid, JSON: payload, CreatedAt: time.Now()})
}

// Get returns one replay.
func (s *Service) Get(ctx context.Context, uuid string) (*Record, error) {
	return s.store.Get(ctx, uuid)
}

// List returns recent replays.
func (s *Service) List(ctx context.Context, limit int) ([]Record, error) {
	return s.store.List(ctx, limit)
}
