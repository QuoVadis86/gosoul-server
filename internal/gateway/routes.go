package gateway

import (
	"fmt"
	"strings"
)

// RouteTable decides which hostnames the proxy intercepts.
type RouteTable struct {
	Domains []string
}

// IsRouted reports whether host (without port) belongs to a proxied domain.
func (t *RouteTable) IsRouted(host string) bool {
	for _, d := range t.Domains {
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

// DefaultDomains is the set of Majsoul server hostnames.
var DefaultDomains = []string{
	"maj-soul.com",
	"mahjongsoul.com",
	"catmajsoul.com",
	"catmjstudio.com",
}

// routesResponse answers the client's route polling with the local lobby.
func routesResponse(lobbyAddr string) []byte {
	host := strings.Split(lobbyAddr, ":")[0]
	port := "8441"
	if i := strings.LastIndex(lobbyAddr, ":"); i >= 0 {
		port = lobbyAddr[i+1:]
	}
	return []byte(fmt.Sprintf(
		`{"data":{"routes":[{"id":"route-1","domain":"%s:%s","ssl":false,"state":"idle","level":1,"order":1,"name":"gosoul"}]}}`,
		host, port))
}

func upgradeResponse() []byte {
	return []byte(`{"data":{"upgrade_list":[]}}`)
}

func announceResponse() []byte {
	return []byte(`{"data":{"announce_list":[]}}`)
}
