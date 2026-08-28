package protocol

var xorKeys = [9]byte{0x84, 0x5e, 0x4e, 0x42, 0x39, 0xa2, 0x1f, 0x60, 0x1c}

// XORCodec applies Majsoul's position-dependent XOR scheme to action payloads.
// XOR is symmetric: the same function both encodes and decodes.
func XORCodec(data []byte) []byte {
	base := 23 ^ len(data)
	out := make([]byte, len(data))
	for i, b := range data {
		k := int(xorKeys[i%len(xorKeys)])
		out[i] = b ^ byte((base+5*i+k)&0xff)
	}
	return out
}

// XORCodecInPlace mutates src in place and returns it. Use when the input buffer
// is owned by the caller and safe to overwrite.
func XORCodecInPlace(data []byte) []byte {
	base := 23 ^ len(data)
	for i := range data {
		k := int(xorKeys[i%len(xorKeys)])
		data[i] ^= byte((base + 5*i + k) & 0xff)
	}
	return data
}
