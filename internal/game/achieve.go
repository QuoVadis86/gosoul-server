package game

import "context"

// Achievements is the outbound hook a session calls when a round produces
// progression. It is nil-safe: sessions without a wired store simply skip.
type Achievements interface {
	// Increment advances the given achievement by delta for the account.
	Increment(ctx context.Context, accountID int64, achieveID, delta int64) error
}

// noopAchievements is the default hook doing nothing.
type noopAchievements struct{}

func (noopAchievements) Increment(context.Context, int64, int64, int64) error { return nil }

// PlayedGame is the achievement id credited for completing one round.
const PlayedGame = 1000
