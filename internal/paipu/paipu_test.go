package paipu

import (
	"context"
	"testing"
)

type memStore struct {
	m map[string]Record
}

func newMem() *memStore { return &memStore{m: map[string]Record{}} }

func (s *memStore) Save(_ context.Context, r Record) error {
	s.m[r.UUID] = r
	return nil
}
func (s *memStore) Get(_ context.Context, uuid string) (*Record, error) {
	r, ok := s.m[uuid]
	if !ok {
		return nil, ErrNotFound
	}
	return &r, nil
}
func (s *memStore) List(_ context.Context, limit int) ([]Record, error) {
	var out []Record
	for _, r := range s.m {
		out = append(out, r)
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func TestPaipuLifecycle(t *testing.T) {
	svc := New(newMem())
	if err := svc.Save(context.Background(), "g-1", `{"scores":[25000,29000,20000,26000]}`); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get(context.Background(), "g-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != "g-1" || got.JSON == "" {
		t.Fatalf("bad record: %+v", got)
	}
	list, err := svc.List(context.Background(), 10)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v (%d)", err, len(list))
	}
}
