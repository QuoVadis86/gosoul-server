package protocol

import "errors"

// MsgType is the first byte of every Majsoul wire frame.
type MsgType byte

const (
	// MsgNotify is a server→client push; no msgId, no correlation.
	MsgNotify MsgType = 0x01
	// MsgRequest is a client→server call carrying a msgId.
	MsgRequest MsgType = 0x02
	// MsgResponse is a server→client reply carrying the caller's msgId.
	MsgResponse MsgType = 0x03
)

// ErrShortFrame is returned when a frame is too small to be valid.
var ErrShortFrame = errors.New("protocol: frame too short")

// Frame is a decoded wire frame.
type Frame struct {
	Type MsgType
	// MsgID is only meaningful for Request and Response frames.
	MsgID uint16
	// Name is the wrapper `name` field: a method like ".lq.Lobby.login" or a
	// notify name like ".lq.ActionPrototype".
	Name string
	// Data is the wrapper `data` field: raw bytes of the typed payload.
	Data []byte
}

// EncodeRequestLE builds a Request frame with little-endian msgID (Majsoul wire order).
func EncodeRequestLE(msgID uint16, method string, payload []byte) []byte {
	f := make([]byte, 0, 3+len(method)+len(payload)+16)
	f = append(f, byte(MsgRequest))
	f = append(f, byte(msgID), byte(msgID>>8))
	f = appendFramePart(f, method, payload)
	return f
}

// EncodeResponse builds a Response frame: [0x03][msgID u16LE][wrapper(empty name)].
func EncodeResponse(msgID uint16, payload []byte) []byte {
	f := make([]byte, 0, 3+len(payload)+8)
	f = append(f, byte(MsgResponse))
	f = append(f, byte(msgID), byte(msgID>>8))
	f = appendFramePart(f, "", payload)
	return f
}

// EncodeNotify builds a Notify frame: [0x01][wrapper].
func EncodeNotify(name string, payload []byte) []byte {
	f := make([]byte, 0, 1+len(name)+len(payload)+16)
	f = append(f, byte(MsgNotify))
	f = appendFramePart(f, name, payload)
	return f
}

// EncodeActionNotify builds the XOR-wrapped ActionPrototype notify used for battle pushes.
func EncodeActionNotify(actionName string, actionData []byte, step uint32) []byte {
	return EncodeNotify(".lq.ActionPrototype", EncodeActionPrototype(actionName, actionData, step))
}

// DecodeFrame parses one complete wire frame into a Frame.
func DecodeFrame(frame []byte) (*Frame, error) {
	if len(frame) < 2 {
		return nil, ErrShortFrame
	}
	m := &Frame{Type: MsgType(frame[0])}
	rest := frame[1:]

	if m.Type == MsgRequest || m.Type == MsgResponse {
		if len(rest) < 2 {
			return nil, ErrShortFrame
		}
		m.MsgID = uint16(rest[0]) | uint16(rest[1])<<8
		rest = rest[2:]
	}

	name, data, err := decodeWrapper(rest)
	if err != nil {
		return nil, err
	}
	m.Name = name
	m.Data = data
	return m, nil
}
