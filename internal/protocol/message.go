package protocol

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Message wraps a dynamic protobuf message with typed field accessors,
// mirroring the ergonomics of the JavaScript reference implementation without
// sacrificing reflection.
type Message struct {
	ref protoreflect.Message
}

// String returns a string field, or "" when absent or of another kind.
func (m *Message) String(field string) string {
	fd := m.fd(field)
	if fd == nil || fd.Kind() != protoreflect.StringKind {
		return ""
	}
	return m.ref.Get(fd).String()
}

// Uint32 returns a numeric field value.
func (m *Message) Uint32(field string) uint32 {
	fd := m.fd(field)
	if fd == nil {
		return 0
	}
	switch fd.Kind() {
	case protoreflect.Uint32Kind, protoreflect.Sint32Kind, protoreflect.Fixed32Kind, protoreflect.Sfixed32Kind:
		return uint32(m.ref.Get(fd).Uint())
	case protoreflect.Int32Kind:
		return uint32(m.ref.Get(fd).Int())
	}
	return 0
}

// Bool returns a boolean field value.
func (m *Message) Bool(field string) bool {
	fd := m.fd(field)
	return fd != nil && fd.Kind() == protoreflect.BoolKind && m.ref.Get(fd).Bool()
}

// HasField reports whether the field is present.
func (m *Message) HasField(field string) bool {
	fd := m.fd(field)
	return fd != nil && m.ref.Has(fd)
}

// SetString sets a string field when it exists.
func (m *Message) SetString(field, value string) {
	fd := m.fd(field)
	if fd == nil || fd.Kind() != protoreflect.StringKind {
		return
	}
	m.ref.Set(fd, protoreflect.ValueOfString(value))
}

// SetUint32 sets a numeric field when it exists.
func (m *Message) SetUint32(field string, value uint32) {
	fd := m.fd(field)
	if fd == nil {
		return
	}
	m.ref.Set(fd, protoreflect.ValueOfUint64(uint64(value)))
}

// SetBool sets a boolean field when it exists.
func (m *Message) SetBool(field string, value bool) {
	fd := m.fd(field)
	if fd == nil || fd.Kind() != protoreflect.BoolKind {
		return
	}
	m.ref.Set(fd, protoreflect.ValueOfBool(value))
}

// Unmarshal fills the message from raw wire bytes.
func (m *Message) Unmarshal(data []byte) error {
	return proto.Unmarshal(data, m.ref.Interface())
}

// Marshal serializes the message to wire bytes.
func (m *Message) Marshal() ([]byte, error) {
	return proto.Marshal(m.ref.Interface())
}

// MarshalJSON renders the message as JSON (logs, admin inspection).
func (m *Message) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(m.ref.Interface())
}

func (m *Message) fd(name string) protoreflect.FieldDescriptor {
	return m.ref.Descriptor().Fields().ByName(protoreflect.Name(name))
}
