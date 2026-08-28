package protocol

import (
	"bytes"
	"testing"
)

func TestXORCodec(t *testing.T) {
	cases := []struct {
		name  string
		input []byte
		want  string
	}{
		{"empty", []byte{}, ""},
		{"v1", []byte{0x0a, 0x03, 0x61}, "92740d"},
		{"v2", []byte{0x0a, 0x01, 0x31, 0x12, 0x01, 0x37, 0x18, 0x01, 0x01}, "a880477d6aee43a063"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := XORCodec(c.input)
			if hex := hexString(got); hex != c.want {
				t.Fatalf("XORCodec mismatch: got %s want %s", hex, c.want)
			}
			// XOR is symmetric.
			back := XORCodec(got)
			if !bytes.Equal(back, c.input) {
				t.Fatalf("XORCodec not symmetric: %x -> %x", c.input, back)
			}
		})
	}
}

func TestEncodeRequestLEMatchesReference(t *testing.T) {
	// Expected frame from the JS reference implementation:
	// [0x02][msgId 0x0001 LE][wrapper name=".lq.Route.heartbeat" data=0800]
	got := EncodeRequestLE(1, ".lq.Route.heartbeat", []byte{0x08, 0x00})
	want := "0201000a132e6c712e526f7574652e68656172746265617412020800"
	if hexString(got) != want {
		t.Fatalf("request frame mismatch:\n got %s\nwant %s", hexString(got), want)
	}
}

func TestDecodeFrame(t *testing.T) {
	frame, err := DecodeFrame(mustHex("0201000a132e6c712e526f7574652e68656172746265617412020800"))
	if err != nil {
		t.Fatal(err)
	}
	if frame.Type != MsgRequest {
		t.Fatalf("type = %d, want %d", frame.Type, MsgRequest)
	}
	if frame.MsgID != 1 {
		t.Fatalf("msgid = %d, want 1", frame.MsgID)
	}
	if frame.Name != ".lq.Route.heartbeat" {
		t.Fatalf("name = %q", frame.Name)
	}
	if !bytes.Equal(frame.Data, []byte{0x08, 0x00}) {
		t.Fatalf("data = %x", frame.Data)
	}
}

func TestEncodeActionNotifyMatchesReference(t *testing.T) {
	got := EncodeActionNotify("ActionNewRound", []byte{0x08, 0x00, 0x12, 0x02, 0x31, 0x6d}, 3)
	want := "010a132e6c712e416374696f6e50726f746f74797065121c0a0e416374696f6e4e6577526f756e6412086e58523759472b681803"
	if hexString(got) != want {
		t.Fatalf("action notify mismatch:\n got %s\nwant %s", hexString(got), want)
	}
}

func TestDecodeNotifyAction(t *testing.T) {
	frame, err := DecodeFrame(mustHex("010a132e6c712e416374696f6e50726f746f74797065121c0a0e416374696f6e4e6577526f756e6412086e58523759472b681803"))
	if err != nil {
		t.Fatal(err)
	}
	if frame.Type != MsgNotify {
		t.Fatalf("type = %d", frame.Type)
	}
	if frame.Name != ".lq.ActionPrototype" {
		t.Fatalf("name = %q", frame.Name)
	}
}

func hexString(b []byte) string {
	const d = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, c := range b {
		out[i*2] = d[c>>4]
		out[i*2+1] = d[c&0xf]
	}
	return string(out)
}

func mustHex(s string) []byte {
	var out []byte
	for i := 0; i < len(s); i += 2 {
		var b byte
		for _, c := range []byte(s[i : i+2]) {
			b <<= 4
			switch {
			case c >= '0' && c <= '9':
				b |= c - '0'
			case c >= 'a' && c <= 'f':
				b |= c - 'a' + 10
			}
		}
		out = append(out, b)
	}
	return out
}
