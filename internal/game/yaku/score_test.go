package yaku

import "testing"

func TestBasePointsSubMangan(t *testing.T) {
	cases := []struct {
		han, fu int
		want    int
	}{
		{1, 30, 240},
		{2, 30, 480},
		{3, 30, 960},
		{4, 30, 1920},
		{4, 40, 2000},
		{4, 50, 2000},
	}
	for _, c := range cases {
		if got := BasePoints(c.han, c.fu); got != c.want {
			t.Errorf("BasePoints(%d,%d)=%d want %d", c.han, c.fu, got, c.want)
		}
	}
}

func TestLimitPayment(t *testing.T) {
	cases := []struct {
		han   int
		nd, d int
	}{
		{5, 8000, 12000},
		{6, 12000, 18000},
		{8, 16000, 24000},
		{11, 24000, 36000},
		{13, 32000, 48000},
	}
	for _, c := range cases {
		nd, d := LimitPayment(c.han)
		if nd != c.nd || d != c.d {
			t.Errorf("LimitPayment(%d)=(%d,%d) want (%d,%d)", c.han, nd, d, c.nd, c.d)
		}
	}
}

func TestRonPayment(t *testing.T) {
	// 1 han 30 fu: base 240; non-dealer ron 4*240=960→1000, dealer ron 6*240=1440→1500
	if got := RonPayment(1, 30, false); got != 1000 {
		t.Errorf("non-dealer ron = %d want 1000", got)
	}
	if got := RonPayment(1, 30, true); got != 1500 {
		t.Errorf("dealer ron = %d want 1500", got)
	}
	// 3 han 40 fu: base 1280 → non-dealer 4x=5120→5200
	if got := RonPayment(3, 40, false); got != 5200 {
		t.Errorf("3han40 ron = %d want 5200", got)
	}
	// 5 han: mangan 8000; dealer 12000
	if got := RonPayment(5, 30, false); got != 8000 {
		t.Errorf("mangan ron = %d want 8000", got)
	}
	if got := RonPayment(5, 30, true); got != 12000 {
		t.Errorf("mangan dealer ron = %d want 12000", got)
	}
}

func TestTsumoPayment(t *testing.T) {
	// 1 han 30 fu non-dealer tsumo: dealer 2*240=480, others 240
	d, nd := TsumoPayment(1, 30, false)
	if d != 500 || nd != 300 {
		t.Errorf("non-dealer tsumo: d=%d nd=%d, want 500/300 (rounded up)", d, nd)
	}
	// dealer tsumo: each pays 480→500
	d, nd = TsumoPayment(1, 30, true)
	if d != 0 || nd != 500 {
		t.Errorf("dealer tsumo: d=%d nd=%d, want 0/500", d, nd)
	}
	// mangan non-dealer tsumo: 2000/4000
	d, nd = TsumoPayment(5, 30, false)
	if d != 4000 || nd != 2000 {
		t.Errorf("mangan tsumo: d=%d nd=%d, want 4000/2000", d, nd)
	}
	// mangan dealer tsumo: 4000 each
	d, nd = TsumoPayment(5, 30, true)
	if d != 0 || nd != 4000 {
		t.Errorf("mangan dealer tsumo: d=%d nd=%d, want 0/4000", d, nd)
	}
}

func TestRoundUp(t *testing.T) {
	cases := map[int]int{240: 300, 480: 500, 960: 1000, 5120: 5200, 2000: 2000}
	for in, want := range cases {
		if got := roundUp(in); got != want {
			t.Errorf("roundUp(%d)=%d want %d", in, got, want)
		}
	}
}
