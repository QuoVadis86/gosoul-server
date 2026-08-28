package gacha

import (
	"context"
	"math/rand"
	"testing"
)

type fakeDrawer struct {
	diamond int64
	granted []int64
}

func (f *fakeDrawer) Balance(_ context.Context, _ int64) (int64, error) { return f.diamond, nil }
func (f *fakeDrawer) Charge(_ context.Context, _, delta int64) error {
	if f.diamond+delta < 0 {
		return ErrNotEnough
	}
	f.diamond += delta
	return nil
}
func (f *fakeDrawer) GrantCharacter(_ context.Context, _, id int64) error {
	f.granted = append(f.granted, id)
	return nil
}

func TestOpenPaysAndGrants(t *testing.T) {
	draw := &fakeDrawer{diamond: 1000}
	svc := New(draw, Pool{Characters: []int64{200001, 200002, 200003}}, rand.New(rand.NewSource(1)))

	res, err := svc.Open(context.Background(), 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ResultList) != 3 {
		t.Fatalf("result list = %d, want 3", len(res.ResultList))
	}
	for _, id := range res.ResultList {
		if id != 200001 && id != 200002 && id != 200003 {
			t.Fatalf("unexpected char %d", id)
		}
	}
	if draw.diamond != 700 {
		t.Fatalf("diamond after = %d, want 700", draw.diamond)
	}
	if len(draw.granted) != 3 {
		t.Fatalf("granted = %d, want 3", len(draw.granted))
	}
}

func TestOpenInsufficientFunds(t *testing.T) {
	draw := &fakeDrawer{diamond: 50}
	svc := New(draw, Pool{Characters: []int64{200001}}, rand.New(rand.NewSource(2)))
	if _, err := svc.Open(context.Background(), 1, 1); err != ErrNotEnough {
		t.Fatalf("open = %v, want ErrNotEnough", err)
	}
	if len(draw.granted) != 0 {
		t.Fatal("no characters should be granted on failure")
	}
}

func TestOpenSingleDefault(t *testing.T) {
	draw := &fakeDrawer{diamond: 100}
	svc := New(draw, Pool{}, rand.New(rand.NewSource(3)))
	res, err := svc.Open(context.Background(), 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ResultList) != 1 || res.ResultList[0] != 200001 {
		t.Fatalf("default single draw = %v", res.ResultList)
	}
}
