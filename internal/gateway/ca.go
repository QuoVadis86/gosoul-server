// Package gateway is the MITM entry point that makes untouched Majsoul
// clients connect to the local servers. It owns a CA, terminates TLS for the
// game domains (maj-soul.com / mahjongsoul.com / ...) and routes those
// requests to the Lobby/Game/Resource servers while passing everything else
// through untouched.
package gateway

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"sync"
	"time"

	"github.com/elazarl/goproxy"
)

// CA loads (or creates) the root CA used to sign per-host certificates.
// Once the CA is trusted by the player's OS (installed via
// `gosoul ca install`), the intercepted game domains verify cleanly.
type CA struct {
	mu     sync.Mutex
	cert   *x509.Certificate
	key    *ecdsa.PrivateKey
	der    []byte
	pem    []byte
	keyPEM []byte
	hosts  map[string]*tls.Certificate
}

// SetAsGoproxyGlobal installs this CA as the signing key elazarl/goproxy uses
// for MITM interception, so intercepted clients verify against our CA.
func (c *CA) SetAsGoproxyGlobal() error {
	kp, err := tls.X509KeyPair(c.pem, c.keyPEM)
	if err != nil {
		return err
	}
	goproxy.GoproxyCa = kp
	return nil
}

// LoadOrCreateCA reads the CA from disk or generates a fresh one.
func LoadOrCreateCA(certPath, keyPath string) (*CA, error) {
	if certBytes, err := os.ReadFile(certPath); err == nil {
		if keyBytes, err := os.ReadFile(keyPath); err == nil {
			if c, err := parseCA(certBytes, keyBytes); err == nil {
				return c, nil
			}
		}
	}
	return createCA(certPath, keyPath)
}

func parseCA(certBytes, keyBytes []byte) (*CA, error) {
	block, _ := pem.Decode(certBytes)
	if block == nil {
		return nil, fmt.Errorf("gateway: malformed CA cert")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	keyBlock, _ := pem.Decode(keyBytes)
	if keyBlock == nil {
		return nil, fmt.Errorf("gateway: malformed CA key")
	}
	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, err
	}
	return &CA{cert: cert, key: key, der: cert.Raw, pem: certBytes, keyPEM: keyBytes, hosts: make(map[string]*tls.Certificate)}, nil
}

func createCA(certPath, keyPath string) (*CA, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	tpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "gosoul private server CA"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.AddDate(10, 0, 0),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dirOf(certPath), 0o755); err != nil {
		return nil, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return nil, err
	}
	return &CA{cert: cert, key: key, der: der, pem: certPEM, keyPEM: keyPEM, hosts: make(map[string]*tls.Certificate)}, nil
}

// CertFor signs a certificate for the given hostname on demand and caches it.
func (c *CA) CertFor(host string) (*tls.Certificate, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cert, ok := c.hosts[host]; ok {
		return cert, nil
	}
	now := time.Now()
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}
	tpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: host},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.AddDate(1, 0, 0),
		DNSNames:     []string{host},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	hostKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, c.cert, &hostKey.PublicKey, c.key)
	if err != nil {
		return nil, err
	}
	cert := &tls.Certificate{
		Certificate: [][]byte{der, c.der},
		PrivateKey:  hostKey,
	}
	c.hosts[host] = cert
	return cert, nil
}

// Public returns the DER bytes of the CA certificate (for trust installation).
func (c *CA) Public() []byte { return c.der }

func dirOf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[:i]
		}
	}
	return "."
}

// SplitHost strips an optional :port suffix from a host header value.
func SplitHost(hostport string) string {
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		return h
	}
	return hostport
}
