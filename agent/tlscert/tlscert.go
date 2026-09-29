// Package tlscert fournit le certificat HTTPS de l'agent. Seul le hub s'y
// connecte : il mémorise l'empreinte du certificat à l'appairage puis la
// vérifie à chaque connexion (comme SSH). Aucune autorité, rien à installer.
package tlscert

import (
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
	"os"
	"path/filepath"
	"time"
)

// Le certificat ne dépend ni de l'adresse ni du nom du PC : il ne change jamais,
// sinon le hub croirait parler à un autre PC.
const validity = 20 * 365 * 24 * time.Hour

// DefaultDir : %APPDATA%\deckpad sous Windows, ~/.config/deckpad sous Linux.
func DefaultDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "deckpad"), nil
}

// Open charge le certificat de dir (agent.crt, agent.key) ou le crée au premier lancement.
func Open(dir string) (*tls.Certificate, error) {
	certPath, keyPath := filepath.Join(dir, "agent.crt"), filepath.Join(dir, "agent.key")
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err == nil {
		return &cert, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("certificat HTTPS : %w", err)
	}
	certPEM, keyPEM, err := create(time.Now())
	if err != nil {
		return nil, fmt.Errorf("certificat HTTPS : %w", err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	// La clé d'abord : un certificat sans sa clé serait inutilisable au prochain lancement.
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return nil, err
	}
	cert, err = tls.X509KeyPair(certPEM, keyPEM)
	return &cert, err
}

// TLSConfig sert toujours le même certificat.
func TLSConfig(cert *tls.Certificate) *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{*cert}}
}

func create(now time.Time) (certPEM, keyPEM []byte, err error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	name := "deckpad"
	if host, err := os.Hostname(); err == nil {
		name = "deckpad (" + host + ")"
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
	if err != nil {
		return nil, nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: name},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(validity),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), nil
}
