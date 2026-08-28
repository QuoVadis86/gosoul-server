// Package yaku implements riichi mahjong hand decomposition, yaku evaluation
// and score calculation. It is game-flow agnostic so both the game engine and
// unit scenarios can reuse it.
package yaku

import "strings"

// T is a canonical tile: rank+suit, e.g. "1m", "9p", "5s", "1z".."7z".
// Red fives are "0m"/"0p"/"0s": they decompose as plain five but score as
// one extra dora.
type T string

func (t T) Suit() byte { return t[len(t)-1] }

// Rank returns the numeric rank; red five normalizes to five.
func (t T) Rank() int {
	r := t[0]
	if r == '0' {
		r = '5'
	}
	return int(r - '0')
}

// IsRed reports whether the tile is a red five.
func (t T) IsRed() bool { return t[0] == '0' }

// Normal strips the red marker (0m→5m).
func (t T) Normal() T {
	if t[0] == '0' {
		return T("5" + t[1:])
	}
	return t
}

// IsHonor reports wind/dragon (suit z).
func (t T) IsHonor() bool { return t.Suit() == 'z' }

// IsTerminal reports rank 1 or 9 in a numbered suit.
func (t T) IsTerminal() bool {
	return !t.IsHonor() && (t.Rank() == 1 || t.Rank() == 9)
}

// IsSimple reports rank 2..8 in a numbered suit.
func (t T) IsSimple() bool { return !t.IsHonor() && !t.IsTerminal() }

// Next returns the tile one rank higher, wrapping per Majsoul dora rules:
// 1z(E)→2z(S)→3z(W)→4z(N)→5z(haku)→6z(hatsu)→7z(chun)→1z.
func (t T) Next() T {
	n := t.Normal()
	if n.IsHonor() {
		r := n.Rank()
		if r == 7 {
			return "1z"
		}
		return T(string(rune('1'+r)) + "z")
	}
	r := n.Rank()
	if r == 9 {
		return T("1" + string(n.Suit()))
	}
	return T(string(rune('1'+r)) + string(n.Suit()))
}

// index maps a tile to its 0..33 count slot (9m,9p,9s then 7 honors).
func (t T) index() int {
	switch t.Suit() {
	case 'm':
		return t.Rank() - 1
	case 'p':
		return 9 + t.Rank() - 1
	case 's':
		return 18 + t.Rank() - 1
	default:
		return 27 + t.Rank() - 1
	}
}

func tileAt(idx int) T {
	switch {
	case idx < 9:
		return T(string(rune('1'+idx)) + "m")
	case idx < 18:
		return T(string(rune('1'+idx-9)) + "p")
	case idx < 27:
		return T(string(rune('1'+idx-18)) + "s")
	default:
		return T(string(rune('1'+idx-27)) + "z")
	}
}

// ParseTiles converts a compact string like "123m456p789s11z" into tiles.
// Unknown characters are ignored, so "1 2 3m" also works.
func ParseTiles(s string) []T {
	var out []T
	suit := byte('m')
	var digits []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
			digits = append(digits, c)
		case c == 'm' || c == 'p' || c == 's' || c == 'z':
			suit = c
			for _, d := range digits {
				out = append(out, T(string(d)+string(suit)))
			}
			digits = digits[:0]
		}
	}
	return out
}

// CountTiles tallies a hand/slice into the 34-slot count form (red fives
// contribute to their plain-five slot only for decomposition).
func CountTiles(hand []T) [34]int {
	var c [34]int
	for _, t := range hand {
		c[t.Normal().index()]++
	}
	return c
}

// SortTiles orders tiles canonically: suits then ranks, honors last, red
// fives first within their rank.
func SortTiles(hand []T) {
	for i := 1; i < len(hand); i++ {
		for j := i; j > 0 && lessTile(hand[j], hand[j-1]); j-- {
			hand[j], hand[j-1] = hand[j-1], hand[j]
		}
	}
}

func lessTile(a, b T) bool {
	ai, bi := a.index(), b.index()
	if ai != bi {
		return ai < bi
	}
	return a.IsRed() && !b.IsRed()
}

// TileString renders tiles space-separated for logs/diagnostics.
func TileString(hand []T) string {
	var b strings.Builder
	for i, t := range hand {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(string(t))
	}
	return b.String()
}
