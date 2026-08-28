package yaku

// CountDora returns the number of dora-value tiles in a hand. A tile is worth
// one dora when it is the immediate successor of an indicator (the indicator
// shows the tile whose mirror is dora). Red fives count one extra.
func CountDora(hand []T, indicators []T) int {
	count := 0
	for _, t := range hand {
		if t.IsRed() {
			count++
		}
		for _, ind := range indicators {
			if t.Normal() == ind.Next() {
				count++
			}
		}
	}
	return count
}

// AddDora appends the dora fan entries to a fan result.
func AddDora(h int) Fan {
	if h <= 0 {
		return Fan{}
	}
	return Fan{Name: "dora", Han: h}
}
