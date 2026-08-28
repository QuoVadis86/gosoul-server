package protocol

import (
	"encoding/base64"
	"errors"

	"google.golang.org/protobuf/encoding/protowire"
)

// ErrBadWrapper is returned when a wrapper protobuf cannot be parsed.
var ErrBadWrapper = errors.New("protocol: malformed wrapper")

// appendFramePart encodes the wrapper {name=1, data=2} fields.
func appendFramePart(dst []byte, name string, data []byte) []byte {
	if len(name) > 0 {
		dst = protowire.AppendTag(dst, 1, protowire.BytesType)
		dst = protowire.AppendString(dst, name)
	}
	if len(data) > 0 {
		dst = protowire.AppendTag(dst, 2, protowire.BytesType)
		dst = protowire.AppendBytes(dst, data)
	}
	return dst
}

// EncodeActionPrototype encodes lq.ActionPrototype {step=1, name=2,
// data=3(base64(xor(payload)))} per the liqi.proto definition.
func EncodeActionPrototype(actionName string, actionData []byte, step uint32) []byte {
	payload := append([]byte(nil), actionData...)
	XORCodecInPlace(payload)
	b64 := base64.StdEncoding.EncodeToString(payload)

	f := make([]byte, 0, len(actionName)+len(b64)+16)
	f = protowire.AppendTag(f, 1, protowire.VarintType)
	f = protowire.AppendVarint(f, uint64(step))
	f = protowire.AppendTag(f, 2, protowire.BytesType)
	f = protowire.AppendString(f, actionName)
	f = protowire.AppendTag(f, 3, protowire.BytesType)
	f = protowire.AppendString(f, b64)
	return f
}

func decodeWrapper(buf []byte) (name string, data []byte, err error) {
	for len(buf) > 0 {
		num, typ, n := protowire.ConsumeTag(buf)
		if n <= 0 {
			return "", nil, ErrBadWrapper
		}
		buf = buf[n:]

		switch typ {
		case protowire.BytesType:
			v, m := protowire.ConsumeBytes(buf)
			if m < 0 {
				return "", nil, ErrBadWrapper
			}
			buf = buf[m:]
			switch num {
			case 1:
				name = string(v)
			case 2:
				data = append([]byte(nil), v...)
			}
		case protowire.VarintType:
			_, m := protowire.ConsumeVarint(buf)
			if m < 0 {
				return "", nil, ErrBadWrapper
			}
			buf = buf[m:]
		default:
			return "", nil, ErrBadWrapper
		}
	}
	return name, data, nil
}
