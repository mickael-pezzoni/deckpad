// Package tlscert fournit le HTTPS du hub : une petite autorité (CA) propre à
// ce hub, que la tablette installe une fois, et un certificat serveur signé par
// elle pour les adresses de la machine.
//
// La CA ne peut signer que des adresses locales (contraintes de nom) : même si
// sa clé était volée, elle ne servirait pas à se faire passer pour un autre site.
package tlscert

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

const (
	caValidity   = 20 * 365 * 24 * time.Hour
	leafValidity = 397 * 24 * time.Hour // plafond accepté par les navigateurs
	renewBefore  = 30 * 24 * time.Hour
)

// Plages d'adresses que la CA a le droit de signer : réseau local, VPN type
// Tailscale (100.64/10) et la machine elle-même.
var permittedRanges = mustCIDRs(
	"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "100.64.0.0/10",
	"169.254.0.0/16", "127.0.0.0/8", "fc00::/7", "fe80::/10", "::1/128",
)

// Authority garde la CA et le certificat serveur courant.
type Authority struct {
	ca     *x509.Certificate
	caKey  crypto.Signer
	now    func() time.Time
	ifaces func() []net.IP

	mu   sync.Mutex
	leaf *tls.Certificate
	ips  []net.IP // adresses couvertes par leaf
}

// Open charge la CA de dir (ca.crt, ca.key) ou la crée au premier lancement.
func Open(dir string) (*Authority, error) {
	a := &Authority{now: time.Now, ifaces: localIPs}
	certPath, keyPath := filepath.Join(dir, "ca.crt"), filepath.Join(dir, "ca.key")
	ca, key, err := load(certPath, keyPath)
	if errors.Is(err, os.ErrNotExist) {
		ca, key, err = create(a.now())
		if err == nil {
			err = save(dir, certPath, keyPath, ca, key)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("autorité HTTPS : %w", err)
	}
	a.ca, a.caKey = ca, key
	return a, nil
}

// CertDER renvoie le certificat de la CA, à installer sur la tablette.
func (a *Authority) CertDER() []byte { return a.ca.Raw }

// TLSConfig sert le certificat serveur, re-signé tout seul quand la machine change
// d'adresse ou que le certificat approche de sa fin.
func (a *Authority) TLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			var local net.IP
			if hello.Conn != nil {
				if addr, ok := hello.Conn.LocalAddr().(*net.TCPAddr); ok {
					local = addr.IP
				}
			}
			return a.certificate(local)
		},
	}
}

func (a *Authority) certificate(local net.IP) (*tls.Certificate, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.leaf != nil && a.now().Add(renewBefore).Before(a.leaf.Leaf.NotAfter) &&
		(local == nil || !permitted(local) || containsIP(a.ips, local)) {
		return a.leaf, nil
	}
	ips := a.ifaces()
	if local != nil && permitted(local) && !containsIP(ips, local) {
		ips = append(ips, local)
	}
	leaf, err := a.sign(ips)
	if err != nil {
		return nil, err
	}
	a.leaf, a.ips = leaf, ips
	return leaf, nil
}

func (a *Authority) sign(ips []net.IP) (*tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	now := a.now()
	tmpl := &x509.Certificate{
		SerialNumber: serial(),
		Subject:      pkix.Name{CommonName: "deckpad"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(leafValidity),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  ips,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, a.ca, &key.PublicKey, a.caKey)
	if err != nil {
		return nil, err
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	return &tls.Certificate{Certificate: [][]byte{der, a.ca.Raw}, PrivateKey: key, Leaf: leaf}, nil
}

func create(now time.Time) (*x509.Certificate, crypto.Signer, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	name := "deckpad hub"
	if host, err := os.Hostname(); err == nil {
		name = "deckpad hub (" + host + ")"
	}
	tmpl := &x509.Certificate{
		SerialNumber:                serial(),
		Subject:                     pkix.Name{CommonName: name, Organization: []string{"deckpad"}},
		NotBefore:                   now.Add(-time.Hour),
		NotAfter:                    now.Add(caValidity),
		KeyUsage:                    x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid:       true,
		IsCA:                        true,
		MaxPathLenZero:              true,
		PermittedDNSDomainsCritical: true,
		PermittedDNSDomains:         []string{"localhost"},
		PermittedIPRanges:           permittedRanges,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}
	cert, err := x509.ParseCertificate(der)
	return cert, key, err
}

func load(certPath, keyPath string) (*x509.Certificate, crypto.Signer, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, nil, err
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, err
	}
	cb, _ := pem.Decode(certPEM)
	kb, _ := pem.Decode(keyPEM)
	if cb == nil || kb == nil {
		return nil, nil, fmt.Errorf("%s ou %s illisible", certPath, keyPath)
	}
	cert, err := x509.ParseCertificate(cb.Bytes)
	if err != nil {
		return nil, nil, err
	}
	key, err := x509.ParsePKCS8PrivateKey(kb.Bytes)
	if err != nil {
		return nil, nil, err
	}
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, nil, fmt.Errorf("%s : clé inattendue", keyPath)
	}
	return cert, signer, nil
}

func save(dir, certPath, keyPath string, cert *x509.Certificate, key crypto.Signer) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	// La clé d'abord : un certificat sans sa clé serait inutilisable au prochain lancement.
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		return err
	}
	return os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}), 0o644)
}

// localIPs liste les adresses locales de la machine que la CA a le droit de signer.
func localIPs() []net.IP {
	ips := []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok && permitted(n.IP) && !containsIP(ips, n.IP) {
			ips = append(ips, n.IP)
		}
	}
	return ips
}

func permitted(ip net.IP) bool {
	return slices.ContainsFunc(permittedRanges, func(n *net.IPNet) bool { return n.Contains(ip) })
}

func containsIP(ips []net.IP, ip net.IP) bool {
	return slices.ContainsFunc(ips, ip.Equal)
}

func serial() *big.Int {
	n, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
	return n
}

func mustCIDRs(cidrs ...string) []*net.IPNet {
	out := make([]*net.IPNet, len(cidrs))
	for i, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			panic(err)
		}
		out[i] = n
	}
	return out
}
