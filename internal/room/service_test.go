package room

import "testing"

type fakePorts struct{}

func (fakePorts) Account(accountID uint32) (string, uint32) {
	return "Player", 400101
}

func TestRoomLifecycle(t *testing.T) {
	s := New(fakePorts{})

	host := uint32(1)
	guest := uint32(2)

	r := s.Create(host, 4, Mode{Mode: 1}, false)
	if r.ID == 0 {
		t.Fatal("create did not assign room id")
	}
	if got := s.RoomOf(host); got != r.ID {
		t.Fatalf("RoomOf(host) = %d, want %d", got, r.ID)
	}
	if r.Players[0].Nickname != "Player" {
		t.Fatalf("host not seeded: %+v", r.Players)
	}

	joined, err := s.Join(r.ID, guest)
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if len(joined.Players) != 2 {
		t.Fatalf("players = %d, want 2", len(joined.Players))
	}

	if _, err := s.SetReady(r.ID, host, true); err != nil {
		t.Fatalf("ready: %v", err)
	}
	if !joined.Players[0].Ready {
		t.Fatal("host not marked ready")
	}

	filled, err := s.AddRobot(r.ID)
	if err != nil {
		t.Fatalf("robot: %v", err)
	}
	if len(filled.Players) != 3 || !filled.Players[2].Robot {
		t.Fatalf("robot seat wrong: %+v", filled.Players)
	}

	if _, err := s.Start(r.ID); err != nil {
		t.Fatalf("start: %v", err)
	}
	if _, err := s.Join(r.ID, 3); err != ErrStarted {
		t.Fatalf("join started room = %v, want ErrStarted", err)
	}

	if err := s.Leave(r.ID, guest); err != nil {
		t.Fatalf("leave: %v", err)
	}
	if s.RoomOf(guest) != 0 {
		t.Fatal("RoomOf(guest) not cleared")
	}
}

func TestRoomCapacity(t *testing.T) {
	s := New(fakePorts{})
	r := s.Create(1, 2, Mode{}, false)
	for id := uint32(2); id <= 3; id++ {
		_, err := s.Join(r.ID, id)
		if err != nil && err != ErrFull {
			t.Fatalf("join %d: %v", id, err)
		}
		if id == 3 && err != ErrFull {
			t.Fatalf("3rd seat expected full")
		}
	}
}

func TestRoomMissing(t *testing.T) {
	s := New(fakePorts{})
	if _, err := s.Get(99); err != ErrNotFound {
		t.Fatalf("Get(99) = %v, want ErrNotFound", err)
	}
	if _, err := s.SetReady(99, 1, true); err != ErrNotFound {
		t.Fatalf("SetReady(99) = %v, want ErrNotFound", err)
	}
}
