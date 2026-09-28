package tlscert

import (
	"crypto/x509"
	"net"
	"testing"
	"time"
)

func verify(t *testing.T, a *Authority, local net.IP, name string) error {
	t.Helper()
	cert, err := a.certificate(local)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(a.ca)
	_, err = cert.Leaf.Verify(x509.VerifyOptions{DNSName: name, Roots: roots, CurrentTime: a.now()})
	return err
}

func TestOpenReusesAuthority(t *testing.T) {
	dir := t.TempDir()
	a, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !a.ca.Equal(b.ca) {
		t.Fatal("la CA doit être gardée d'un lancement à l'autre")
	}
}

func TestLeafCoversLocalAddresses(t *testing.T) {
	a, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a.ifaces = func() []net.IP { return []net.IP{net.ParseIP("192.168.1.20")} }
	if err := verify(t, a, nil, "192.168.1.20"); err != nil {
		t.Fatal(err)
	}
	// Nouvelle adresse (DHCP, autre réseau) : re-signé sans redémarrer.
	if err := verify(t, a, net.ParseIP("10.0.0.7"), "10.0.0.7"); err != nil {
		t.Fatal(err)
	}
}

func TestAuthorityCannotSignPublicAddresses(t *testing.T) {
	a, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := a.sign([]net.IP{net.ParseIP("8.8.8.8")})
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(a.ca)
	if _, err := leaf.Leaf.Verify(x509.VerifyOptions{DNSName: "8.8.8.8", Roots: roots}); err == nil {
		t.Fatal("une adresse publique ne doit pas être acceptée")
	}
}

func TestLeafRenewedBeforeExpiry(t *testing.T) {
	a, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	first, _ := a.certificate(nil)
	a.now = func() time.Time { return time.Now().Add(leafValidity - renewBefore/2) }
	second, _ := a.certificate(nil)
	if first == second {
		t.Fatal("le certificat doit être renouvelé avant sa fin")
	}
}
