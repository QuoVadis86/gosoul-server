package yaku

// Mentsu is one decomposed set of a winning hand.
type Mentsu struct {
	Shuntsu bool // sequence (123m) vs triplet/quad (555p)
	Tile    T    // low tile for a sequence; the triplet/quad tile otherwise
	N       int  // 3 (koutsu) or 4 (kantsu)
	Open    bool // claimed (heng/pon/daiminkan/kakan) vs concealed
}

// Result is a complete decomposition of one 14-tile hand.
type Result struct {
	Pair  T        // atama (the pair)
	Melds []Mentsu // the four sets (sequences and triplets)
}

// Analyze enumerates every valid decomposition of a 14-tile winning hand.
// hand holds the concealed tiles plus the winning tile: 14 tiles with no open
// melds, or 14-3*len(open) otherwise. Kokushi and chiitoitsu (when no melds
// are open) are returned as their single special result.
func Analyze(hand []T, open []Mentsu) []*Result {
	expect := 14 - 3*len(open)
	if len(hand) != expect {
		return nil
	}
	if len(open) == 0 {
		if isKokushi(hand) {
			return []*Result{{Pair: ""}}
		}
		if r := chiitoitsu(hand); r != nil {
			return []*Result{r}
		}
	}
	counts := CountTiles(hand)
	var out []*Result
	for pair := 0; pair < 34; pair++ {
		if counts[pair] < 2 {
			continue
		}
		counts[pair] -= 2
		var melds []Mentsu
		if splitSets(counts, 4-len(open), &melds) {
			r := &Result{Pair: tileAt(pair)}
			r.Melds = append(r.Melds, open...)
			r.Melds = append(r.Melds, melds...)
			out = append(out, r)
		}
		counts[pair] += 2
	}
	return out
}

func isKokushi(hand []T) bool {
	if len(hand) != 14 {
		return false
	}
	counts := CountTiles(hand)
	pair := false
	for i, n := range counts {
		if !isTerminalIndex(i) {
			if n != 0 {
				return false
			}
			continue
		}
		switch n {
		case 1:
		case 2:
			if pair {
				return false
			}
			pair = true
		default:
			return false
		}
	}
	return pair
}

func isTerminalIndex(i int) bool {
	if i < 27 {
		return i%9 == 0 || i%9 == 8
	}
	return true
}

func chiitoitsu(hand []T) *Result {
	counts := CountTiles(hand)
	pairs := 0
	for _, n := range counts {
		if n != 0 && n != 2 {
			return nil
		}
		if n == 2 {
			pairs++
		}
	}
	if pairs != 7 {
		return nil
	}
	return &Result{Pair: ""}
}

// Tenpai returns the tiles that complete the 13-tile concealed hand given the
// already-claimed open melds. The returned map value is unused for now.
func Tenpai(hand []T, open []Mentsu) map[T]struct{} {
	if len(hand) != 13 {
		return nil
	}
	base := CountTiles(hand)
	var out map[T]struct{}
	for i := 0; i < 34; i++ {
		if base[i] >= 4 {
			continue
		}
		base[i]++
		if len(Analyze(recount(base), open)) > 0 {
			if out == nil {
				out = make(map[T]struct{})
			}
			out[tileAt(i)] = struct{}{}
		}
		base[i]--
	}
	return out
}

// recount rebuilds a tile slice from a count form.
func recount(c [34]int) []T {
	var out []T
	for i, n := range c {
		for j := 0; j < n; j++ {
			out = append(out, tileAt(i))
		}
	}
	return out
}

// splitSets recursively consumes counts into `need` sets. It removes the
// lowest remaining tile first: a sequence when possible (suits only), then a
// triplet. counts is scratch and is restored on backtracking.
func splitSets(counts [34]int, need int, acc *[]Mentsu) bool {
	if need == 0 {
		for i := 0; i < 34; i++ {
			if counts[i] != 0 {
				return false
			}
		}
		return true
	}
	start := 0
	for start < 34 && counts[start] == 0 {
		start++
	}
	if start >= 34 {
		return false
	}
	t := tileAt(start)
	if !t.IsHonor() && start%9 < 7 && counts[start+1] > 0 && counts[start+2] > 0 {
		counts[start]--
		counts[start+1]--
		counts[start+2]--
		*acc = append(*acc, Mentsu{Shuntsu: true, Tile: t, N: 3})
		if splitSets(counts, need-1, acc) {
			return true
		}
		*acc = (*acc)[:len(*acc)-1]
		counts[start]++
		counts[start+1]++
		counts[start+2]++
	}
	if counts[start] >= 3 {
		counts[start] -= 3
		*acc = append(*acc, Mentsu{Shuntsu: false, Tile: t, N: 3})
		if splitSets(counts, need-1, acc) {
			return true
		}
		*acc = (*acc)[:len(*acc)-1]
		counts[start] += 3
	}
	return false
}

// Clone returns a deep copy of a result.
func (r *Result) Clone() *Result {
	n := *r
	n.Melds = append([]Mentsu(nil), r.Melds...)
	return &n
}
