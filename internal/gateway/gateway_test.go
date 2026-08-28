package gateway

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// startProxy boots the real MITM proxy over a httptest server and returns its
// URL and the CA PEM clients must trust for intercepted domains.
func startProxy(t *testing.T) (*url.URL, []byte) {
	t.Helper()
	dir := t.TempDir()
	ca, err := LoadOrCreateCA(filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem"))
	if err != nil {
		t.Fatalf("CA: %v", err)
	}
	srv, err := NewServer(Config{
		CA:           ca,
		Domains:      DefaultDomains,
		LobbyAddr:    "127.0.0.1:8441",
		ResourceAddr: "127.0.0.1:8440",
	}, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	proxyURL, _ := url.Parse(ts.URL)
	pem, _ := os.ReadFile(filepath.Join(dir, "cert.pem"))
	return proxyURL, pem
}

func mitmClient(proxy *url.URL, rootPEM []byte, extra ...*x509.Certificate) *http.Client {
	pool := x509.NewCertPool()
	if len(rootPEM) > 0 {
		pool.AppendCertsFromPEM(rootPEM)
	}
	for _, c := range extra {
		pool.AddCert(c)
	}
	return &http.Client{Transport: &http.Transport{
		Proxy:           http.ProxyURL(proxy),
		TLSClientConfig: &tls.Config{RootCAs: pool},
	}}
}

func mitmGet(t *testing.T, client *http.Client, target string) string {
	t.Helper()
	resp, err := client.Get(target)
	if err != nil {
		t.Fatalf("GET %s via proxy: %v", target, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s status=%d body=%s", target, resp.StatusCode, body)
	}
	return string(body)
}

func TestMITMRoutesAnswered(t *testing.T) {
	proxy, pem := startProxy(t)
	client := mitmClient(proxy, pem)

	body := mitmGet(t, client, "https://route-2.maj-soul.com/api/clientgate/routes")
	if !strings.Contains(body, `"127.0.0.1:8441"`) {
		t.Fatalf("routes payload missing lobby: %s", body)
	}
}

func TestMITMUpgradeInfo(t *testing.T) {
	proxy, pem := startProxy(t)
	client := mitmClient(proxy, pem)

	body := mitmGet(t, client, "https://route-3.maj-soul.com/api/clientgate/upgrade_info")
	if !strings.Contains(body, `"upgrade_list"`) {
		t.Fatalf("upgrade payload: %s", body)
	}
}

func TestNonGameDomainBypassesMitm(t *testing.T) {
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("upstream-ok"))
	}))
	defer upstream.Close()

	proxy, _ := startProxy(t)
	// Trust ONLY the upstream's certificate: if the proxy MITM'd this domain
	// it would present our CA instead and verification would fail.
	client := mitmClient(proxy, nil, upstream.Certificate())
	resp, err := client.Get(upstream.URL + "/x")
	if err != nil {
		t.Fatalf("GET upstream via proxy: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "upstream-ok" {
		t.Fatalf("upstream body: %s", body)
	}
}

func TestNonGameDomainFailsIfMitmed(t *testing.T) {
	// Negative control: trust only our CA; reaching the upstream through the
	// same proxy MUST fail, proving non-game domains are never MITM'd.
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer upstream.Close()

	proxy, pem := startProxy(t)
	client := mitmClient(proxy, pem)
	if resp, err := client.Get(upstream.URL + "/x"); err == nil {
		resp.Body.Close()
		t.Fatal("upstream unexpectedly reachable with our CA: domain was MITM'd")
	}
}
