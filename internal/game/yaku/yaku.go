package yaku

// ctx carries everything yaku evaluation needs beyond the decomposition.
type Ctx struct {
	SeatWind  int // 0=E 1=S 2=W 3=N
	RoundWind int
	Zimo      bool
	Menzen    bool // no open melds
	Riichi    bool
	Daburi    bool
	Ippatsu   bool
	Rinshan   bool // tsumo after a kan
	Chankan   bool // ron on a ron-eligible discard by someone else (抢杠/荣和)
	Honba     int
}

// Fan is a counted yaku contribution.
type Fan struct {
	Name string
	Han  int
}

// FanResult aggregates all counted yaku for a hand.
type FanResult struct {
	Fans    []Fan
	Total   int
	Yakuman bool
}

// Calc evaluates every yaku on a decomposition and returns the combined han.
func Calc(r *Result, hand []T, c *Ctx) *FanResult {
	// chiitoitsu is a special shape: seven pairs (2 han).
	if len(r.Melds) == 0 && isChiitoitsu(hand) {
		return &FanResult{Fans: []Fan{{Name: "chiitoitsu", Han: 2}}, Total: 2}
	}
	// kokushi substitutes a pair-free single result.
	if len(r.Melds) == 0 && isKokushi(hand) {
		return &FanResult{Fans: []Fan{{Name: "kokushi", Han: 13}}, Yakuman: true, Total: 13}
	}
	var out []Fan
	add := func(name string, han int) {
		out = append(out, Fan{Name: name, Han: han})
	}

	openCount := 0
	shuntsuCount := 0
	for _, m := range r.Melds {
		if m.Open {
			openCount++
		}
		if m.Shuntsu {
			shuntsuCount++
		}
	}
	menzen := openCount == 0

	// yakuman first: anything that is already counted as yakuman suppresses
	// regular yaku and the rest of the evaluation.
	if ym := yakuman(r, hand, c); ym != "" {
		return &FanResult{Fans: []Fan{{Name: ym, Han: 13}}, Yakuman: true, Total: 13}
	}

	if c.Riichi {
		if c.Daburi {
			add("daburi", 2)
		} else {
			add("riichi", 1)
		}
		if c.Ippatsu {
			add("ippatsu", 1)
		}
	}
	if c.Zimo && menzen {
		add("menzenTsumo", 1)
	}
	if c.Rinshan {
		add("rinshan", 1)
	}
	if c.Chankan {
		add("chankan", 1)
	}

	// pinfu: closed, all sequences, non-valued pair, open wait.
	if menzen && !c.Zimo && shuntsuCount == 4 && !isValuelessPair(r.Pair, c) {
		add("pinfu", 1)
	}

	// tanyao: no terminals or honors anywhere.
	if tanyao(r, hand) {
		add("tanyao", 1)
	}

	// yakuhai
	if h := yakuhai(r, c); len(h) > 0 {
		for _, name := range h {
			add(name, 1)
		}
	}

	// Toitoi: every set is a triplet (concealed or open).
	if shuntsuCount == 0 && len(r.Melds) == 4 {
		add("toitoi", 2)
	}

	if chanta(r, hand, false) {
		if menzen {
			add("chanta", 2)
		} else {
			add("chanta", 1)
		}
	}
	if chanta(r, hand, true) {
		if menzen {
			add("junchan", 3)
		} else {
			add("junchan", 2)
		}
	}

	if honitsu(r, hand) {
		if menzen {
			add("honitsu", 3)
		} else {
			add("honitsu", 2)
		}
	}
	if chinitsu(r, hand) {
		if menzen {
			add("chinitsu", 6)
		} else {
			add("chinitsu", 5)
		}
	}

	if sanshokuDoujun(r) {
		if menzen {
			add("sanshoku", 2)
		} else {
			add("sanshokuOpen", 1)
		}
	}
	if ittsu(r, menzen) {
		add("ittsu", ittsuHan(menzen))
	}

	total := 0
	for _, f := range out {
		total += f.Han
	}
	return &FanResult{Fans: out, Total: total}
}

func (c *Ctx) valuedPair(pair T) bool { return pair.IsHonor() }

func isValuelessPair(pair T, c *Ctx) bool {
	if pair.IsHonor() {
		return false
	}
	return pair.IsSimple()
}

func tanyao(r *Result, hand []T) bool {
	if r.Pair.IsHonor() || r.Pair.IsTerminal() {
		return false
	}
	for _, m := range r.Melds {
		if m.Tile.IsHonor() || m.Tile.IsTerminal() {
			return false
		}
		if m.Shuntsu && (m.Tile.Rank() == 1 || m.Tile.Rank() == 7) {
			return false // 123 or 789 contains a terminal
		}
	}
	return true
}

func yakuhai(r *Result, c *Ctx) []string {
	var dragons []string
	var winds []string
	seatTile := T(string(rune('1'+c.SeatWind)) + "z")
	roundTile := T(string(rune('1'+c.RoundWind)) + "z")
	check := func(t T) string {
		switch t {
		case "5z":
			return "haku"
		case "6z":
			return "hatsu"
		case "7z":
			return "chun"
		}
		return ""
	}
	for _, m := range r.Melds {
		if m.Shuntsu || !m.Tile.IsHonor() {
			continue
		}
		if name := check(m.Tile); name != "" {
			dragons = append(dragons, name)
		}
		if m.Tile == seatTile {
			winds = append(winds, "yakuhaiSeat")
		}
		if m.Tile == roundTile {
			winds = append(winds, "yakuhaiRound")
		}
	}
	return append(dragons, winds...)
}

func chanta(r *Result, hand []T, jun bool) bool {
	if jun && r.Pair.IsHonor() {
		return false
	}
	check := func(t T) bool {
		return jun && !t.IsHonor() && t.IsTerminal() || !jun && (t.IsTerminal() || t.IsHonor())
	}
	if !check(r.Pair) {
		return false
	}
	for _, m := range r.Melds {
		if m.Shuntsu {
			if m.Tile.Rank() != 1 && m.Tile.Rank() != 7 {
				return false
			}
			continue
		}
		if !check(m.Tile) {
			return false
		}
	}
	return true
}

func honitsu(r *Result, hand []T) bool {
	suits := map[byte]int{}
	honorCount := 0
	for _, t := range hand {
		if t.IsHonor() {
			honorCount++
		} else {
			suits[t.Suit()]++
		}
	}
	maxSuitCount := 0
	total := 0
	for _, n := range suits {
		if n > maxSuitCount {
			maxSuitCount = n
		}
		total += n
	}
	return honorCount > 0 && maxSuitCount == total
}

func chinitsu(r *Result, hand []T) bool {
	if len(hand) == 0 {
		return false
	}
	suit := hand[0].Suit()
	for _, m := range r.Melds {
		if m.Tile.IsHonor() {
			return false
		}
	}
	for _, t := range hand {
		if t.IsHonor() || t.Suit() != suit {
			return false
		}
	}
	return true
}

// sanshokuDoujun reports three identical sequences across the three numbered
// suits (e.g. 123m 123p 123s), counting each sequence once per suit.
func sanshokuDoujun(r *Result) bool {
	byRank := map[int][]byte{}
	for _, m := range r.Melds {
		if !m.Shuntsu {
			continue
		}
		rank := m.Tile.Rank()
		suit := m.Tile.Suit()
		dup := false
		for _, s := range byRank[rank] {
			if s == suit {
				dup = true
				break
			}
		}
		if !dup {
			byRank[rank] = append(byRank[rank], suit)
		}
	}
	for _, suits := range byRank {
		if len(suits) == 3 {
			return true
		}
	}
	return false
}

func ittsu(r *Result, menzen bool) bool {
	var have123, have456, have789 bool
	for _, m := range r.Melds {
		if !m.Shuntsu {
			continue
		}
		if m.Tile.Rank() == 1 {
			have123 = true
		}
		if m.Tile.Rank() == 4 {
			have456 = true
		}
		if m.Tile.Rank() == 7 {
			have789 = true
		}
	}
	return have123 && have456 && have789
}

func ittsuHan(menzen bool) int {
	if menzen {
		return 2
	}
	return 1
}

func yakuman(r *Result, hand []T, c *Ctx) string {
	// kokushi: all terminals/honors once each plus one duplicate.
	if isKokushi(hand) {
		return "kokushi"
	}
	// suuankou: four concealed triplets.
	if fourConcealedTriplets(r) {
		return "suuankou"
	}
	return ""
}

func isChiitoitsu(hand []T) bool {
	if len(hand) != 14 {
		return false
	}
	counts := CountTiles(hand)
	pairs := 0
	for _, n := range counts {
		switch n {
		case 0, 2:
			if n == 2 {
				pairs++
			}
		default:
			return false
		}
	}
	return pairs == 7
}

func fourConcealedTriplets(r *Result) bool {
	if len(r.Melds) != 4 {
		return false
	}
	for _, m := range r.Melds {
		if m.Shuntsu || m.Open {
			return false
		}
	}
	return true
}
