package yaku

// LimitPayment returns the direct ron/tsumo payment for a limit hand (man and
// above); (nonDealerRon, dealerRon) in points. Below that, payments derive
// from BasePoints.
func LimitPayment(han int) (nonDealerRon, dealerRon int) {
	switch {
	case han >= 13:
		return 32000, 48000
	case han >= 11:
		return 24000, 36000
	case han >= 8:
		return 16000, 24000
	case han >= 6:
		return 12000, 18000
	default:
		return 8000, 12000
	}
}

// BasePoints returns the unrounded base value of a sub-limit hand: the value
// used by RonPayment/TsumoPayment below mangan, where fu×2^(2+han) applies
// with a 2000 cap.
func BasePoints(han, fu int) int {
	if han >= 5 || han >= 4 && fu >= 40 {
		return 2000
	}
	bp := fu * (1 << (2 + han))
	if bp > 2000 {
		bp = 2000
	}
	return bp
}

// roundUp rounds a payment to the nearest 100, upward (riichi rules).
func roundUp(p int) int { return (p + 99) / 100 * 100 }

// RonPayment returns how much the single discarder pays the ron winner, using
// the direct limit table at man and above.
func RonPayment(han, fu int, byDealer bool) int {
	if han >= 5 {
		nd, d := LimitPayment(han)
		if byDealer {
			return d
		}
		return nd
	}
	bp := BasePoints(han, fu)
	if byDealer {
		return roundUp(bp * 6)
	}
	return roundUp(bp * 4)
}

// TsumoPayment splits a tsumo win and rounds each payer up:
// (dealerPays, nonDealerPays). A dealer winner pays nothing itself. Limit
// hands use the direct table: 4000/2000 at mangan, 6000/3000 at haneman, etc.
func TsumoPayment(han, fu int, winnerDealer bool) (dealerPays, nonDealerPays int) {
	if han >= 5 {
		nd, _ := LimitPayment(han)
		if winnerDealer {
			return 0, roundUp(nd / 2)
		}
		return roundUp(nd / 2), roundUp(nd / 4)
	}
	bp := BasePoints(han, fu)
	if winnerDealer {
		return 0, roundUp(bp * 2)
	}
	return roundUp(bp * 2), roundUp(bp)
}

// Fu computes the fu count for a hand. winOnClosed marks a ron win on a
// closed hand (menzen ron +10), instead of a closed tsumo (+2).
func Fu(r *Result, c *ctx, winOnClosed bool) int {
	fu := 20
	if c.Zimo {
		fu += 2
	}
	if winOnClosed {
		fu += 10
	}
	seatTile := T(string(rune('1'+c.SeatWind)) + "z")
	roundTile := T(string(rune('1'+c.RoundWind)) + "z")
	if r.Pair.IsHonor() || r.Pair == seatTile || r.Pair == roundTile {
		fu += 2
	}
	for _, m := range r.Melds {
		if m.Shuntsu {
			continue
		}
		v := 2
		if m.N == 4 {
			v = 8
		}
		if m.Tile.IsTerminal() || m.Tile.IsHonor() {
			v *= 2
		}
		if !m.Open {
			v *= 2
		}
		fu += v
	}
	if fu%10 == 0 {
		return fu
	}
	return (fu + 9) / 10 * 10
}
