package protocol

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/reporter"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

//go:embed data/liqi.proto
var liqiProto []byte

//go:embed data/liqi.json
var liqiJSON []byte

// Route maps a full method name (".lq.Lobby.login") to its message types.
type Route struct {
	Method   string
	ReqType  string
	RespType string
}

// Registry holds the message type table and the method routing table.
type Registry struct {
	types    *protoregistry.Types
	routes   map[string]Route
	notifies map[string]string
}

// Load compiles the embedded liqi.proto and builds the routing table from the
// embedded liqi.json descriptor. Safe to call once at server startup.
func Load() (*Registry, error) {
	compiler := protocompile.Compiler{
		Resolver:       embedResolver{},
		SourceInfoMode: protocompile.SourceInfoNone,
		Reporter: reporter.NewReporter(
			func(err reporter.ErrorWithPos) error { return err },
			nil,
		),
	}
	files, err := compiler.Compile(context.Background(), "liqi.proto")
	if err != nil {
		return nil, fmt.Errorf("compile liqi.proto: %w", err)
	}

	fd := files[0]
	reg := &Registry{
		types:    new(protoregistry.Types),
		routes:   make(map[string]Route),
		notifies: make(map[string]string),
	}
	registerMessages(reg.types, fd.Messages())

	if err := reg.buildRoutes(); err != nil {
		return nil, err
	}
	return reg, nil
}

// NewMessage allocates a dynamic message for a fully-qualified type name such
// as "lq.ResLogin".
func (r *Registry) NewMessage(typeName string) (*Message, error) {
	mt, err := r.types.FindMessageByName(protoreflect.FullName(typeName))
	if err != nil {
		return nil, err
	}
	return &Message{ref: mt.New()}, nil
}

// RouteFor returns the request/response types for a method, if registered.
func (r *Registry) RouteFor(methodName string) (Route, bool) {
	route, ok := r.routes[methodName]
	return route, ok
}

// NotifyTypeFor returns the message type name for a notify name like
// ".lq.NotifyRoomGameStart" when the type is registered.
func (r *Registry) NotifyTypeFor(name string) (string, bool) {
	idx := strings.LastIndex(name, ".")
	if idx < 0 || idx+1 == len(name) {
		return "", false
	}
	t := "lq." + name[idx+1:]
	_, err := r.types.FindMessageByName(protoreflect.FullName(t))
	return t, err == nil
}

type embedResolver struct{}

func (embedResolver) FindFileByPath(path string) (protocompile.SearchResult, error) {
	if path != "liqi.proto" {
		return protocompile.SearchResult{}, errors.New("unknown proto path: " + path)
	}
	return protocompile.SearchResult{Source: bytes.NewReader(liqiProto)}, nil
}

func registerMessages(types *protoregistry.Types, msgs protoreflect.MessageDescriptors) {
	for i := 0; i < msgs.Len(); i++ {
		md := msgs.Get(i)
		if err := types.RegisterMessage(dynamicpb.NewMessageType(md)); err != nil {
			continue
		}
		registerMessages(types, md.Messages())
	}
}

type liqiMethod struct {
	RequestType  string `json:"requestType"`
	ResponseType string `json:"responseType"`
}

type liqiService struct {
	Methods map[string]liqiMethod `json:"methods"`
}

type liqiRoot struct {
	Nested struct {
		Lq struct {
			Nested map[string]liqiService `json:"nested"`
		} `json:"lq"`
	} `json:"nested"`
}

func (r *Registry) buildRoutes() error {
	var root liqiRoot
	if err := json.Unmarshal(liqiJSON, &root); err != nil {
		return fmt.Errorf("parse liqi.json: %w", err)
	}
	for svcName, svc := range root.Nested.Lq.Nested {
		for method, m := range svc.Methods {
			full := ".lq." + svcName + "." + method
			r.routes[full] = Route{
				Method:   full,
				ReqType:  "lq." + m.RequestType,
				RespType: "lq." + m.ResponseType,
			}
		}
	}
	return nil
}
