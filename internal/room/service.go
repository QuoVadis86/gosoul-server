// Package room implements the friend-room domain: create/join/leave, ready
// state, and robot filling. Rooms live in memory for the session lifetime and
// are handed to the game layer once start is requested.
package room

import "errors"

// ErrNotFound is returned when a room does not exist.
var ErrNotFound = errors.New("room: not found")

// ErrFull is returned when a room has reached its player limit.
var ErrFull = errors.New("room: full")

// ErrStarted is returned when join is attempted after the game started.
var ErrStarted = errors.New("room: game already started")

// Mode is the detail-rule bag the client sends at creation.
type Mode struct {
	Mode       uint32         `json:"mode"`
	DetailRule map[string]any `json:"detailRule"`
}

// Player is one seat inside a room.
type Player struct {
	AccountID uint32 `json:"accountId"`
	Nickname  string `json:"nickname"`
	AvatarID  uint32 `json:"avatarId"`
	Ready     bool   `json:"ready"`
	Robot     bool   `json:"robot"`
}

// Room is the aggregate root of the friend-room domain.
type Room struct {
	ID             uint32   `json:"roomId"`
	OwnerID        uint32   `json:"ownerId"`
	State          uint32   `json:"state"`
	Players        []Player `json:"players"`
	MaxPlayerCount uint32   `json:"maxPlayerCount"`
	Mode           Mode     `json:"mode"`
	PublicLive     bool     `json:"publicLive"`
	GameStarted    bool     `json:"gameStarted"`
}

// Ports is the persistence seam the lobby surface drives through the service.
type Ports interface {
	Account(accountID uint32) (nickname string, avatarID uint32)
}

// Service owns room lifecycle over an in-memory table.
type Service struct {
	ports  Ports
	rooms  map[uint32]*Room
	member map[uint32]uint32
	nextID uint32
}

// New wires the service over its ports.
func New(p Ports) *Service {
	return &Service{ports: p, rooms: make(map[uint32]*Room), member: make(map[uint32]uint32), nextID: 1}
}

// Rooms exposes every live room for the surface to scan.
func (s *Service) Rooms() []*Room {
	out := make([]*Room, 0, len(s.rooms))
	for _, r := range s.rooms {
		out = append(out, r)
	}
	return out
}

// RoomOf returns the room id an account currently sits in (0 = none).
func (s *Service) RoomOf(accountID uint32) uint32 {
	return s.member[accountID]
}

// Create opens a new room owned by accountID.
func (s *Service) Create(ownerID uint32, maxPlayers uint32, mode Mode, publicLive bool) *Room {
	nickname, avatarID := s.nickAvatar(ownerID)
	r := &Room{
		ID:             s.nextID,
		OwnerID:        ownerID,
		State:          1,
		Players:        []Player{{AccountID: ownerID, Nickname: nickname, AvatarID: avatarID}},
		MaxPlayerCount: maxPlayers,
		Mode:           mode,
		PublicLive:     publicLive,
	}
	s.nextID++
	s.rooms[r.ID] = r
	s.member[ownerID] = r.ID
	return r
}

func (s *Service) nickAvatar(accountID uint32) (string, uint32) {
	if s.ports != nil {
		return s.ports.Account(accountID)
	}
	return "Player", 400101
}

// Get returns a room by id.
func (s *Service) Get(id uint32) (*Room, error) {
	r, ok := s.rooms[id]
	if !ok {
		return nil, ErrNotFound
	}
	return r, nil
}

// Join adds a player unless the room is full or running.
func (s *Service) Join(roomID, accountID uint32) (*Room, error) {
	r, err := s.Get(roomID)
	if err != nil {
		return nil, err
	}
	if r.GameStarted {
		return nil, ErrStarted
	}
	for _, p := range r.Players {
		if p.AccountID == accountID {
			return r, nil
		}
	}
	if uint32(len(r.Players)) >= r.MaxPlayerCount {
		return nil, ErrFull
	}
	nickname, avatarID := s.nickAvatar(accountID)
	r.Players = append(r.Players, Player{AccountID: accountID, Nickname: nickname, AvatarID: avatarID})
	s.member[accountID] = r.ID
	return r, nil
}

// Leave removes a player; empty rooms are discarded.
func (s *Service) Leave(roomID, accountID uint32) error {
	r, err := s.Get(roomID)
	if err != nil {
		return err
	}
	out := r.Players[:0]
	for _, p := range r.Players {
		if p.AccountID != accountID {
			out = append(out, p)
		}
	}
	r.Players = out
	delete(s.member, accountID)
	if len(r.Players) == 0 {
		delete(s.rooms, roomID)
	}
	return nil
}

// SetReady toggles a player's ready flag.
func (s *Service) SetReady(roomID, accountID uint32, ready bool) (*Room, error) {
	r, err := s.Get(roomID)
	if err != nil {
		return nil, err
	}
	for i := range r.Players {
		if r.Players[i].AccountID == accountID {
			r.Players[i].Ready = ready
			return r, nil
		}
	}
	return nil, ErrNotFound
}

// AddRobot fills an empty seat with a robot player.
func (s *Service) AddRobot(roomID uint32) (*Room, error) {
	r, err := s.Get(roomID)
	if err != nil {
		return nil, err
	}
	if uint32(len(r.Players)) >= r.MaxPlayerCount {
		return nil, ErrFull
	}
	r.Players = append(r.Players, Player{AccountID: robotID(r), Nickname: "Robot", AvatarID: 400101, Robot: true})
	return r, nil
}

func robotID(r *Room) uint32 {
	base := r.ID * 1000
	for i := uint32(1); ; i++ {
		candidate := base + i
		found := false
		for _, p := range r.Players {
			if p.AccountID == candidate {
				found = true
				break
			}
		}
		if !found {
			return candidate
		}
	}
}

// Start marks the room as handed to the game layer.
func (s *Service) Start(roomID uint32) (*Room, error) {
	r, err := s.Get(roomID)
	if err != nil {
		return nil, err
	}
	r.GameStarted = true
	return r, nil
}
