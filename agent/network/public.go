package network

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// L'IP publique vient d'un service externe ; on la garde 10 minutes en cache.
var (
	publicMu  sync.Mutex
	publicIP  string
	publicAt  time.Time
	publicTTL = 10 * time.Minute
)

func PublicIP(ctx context.Context) (string, error) {
	publicMu.Lock()
	defer publicMu.Unlock()
	if publicIP != "" && time.Since(publicAt) < publicTTL {
		return publicIP, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.ipify.org", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return "", err
	}
	ip := strings.TrimSpace(string(b))
	if net.ParseIP(ip) == nil {
		return "", &net.ParseError{Type: "IP address", Text: ip}
	}
	publicIP, publicAt = ip, time.Now()
	return ip, nil
}
