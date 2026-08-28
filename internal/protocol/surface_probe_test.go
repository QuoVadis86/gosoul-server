package protocol

import "testing"

func TestAllResponseTypesInstantiable(t *testing.T) {
	reg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	var missing []string
	for _, m := range reg.Methods() {
		route, _ := reg.RouteFor(m)
		if _, err := reg.NewMessage(route.RespType); err != nil {
			missing = append(missing, route.RespType)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("uninstantiable response types: %v", missing)
	}
}
