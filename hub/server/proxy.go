package server

import (
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/mickael-pezzoni/deckpad/hub/agents"
)

// handleProxy relaie /api/pc/{id}/… vers /api/… de l'agent, avec la clé du hub
// et en vérifiant l'empreinte de son certificat. Les flux SSE passent tels quels.
func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	if !s.tabletPaired(r) {
		reason(w, http.StatusUnauthorized, "tablet")
		return
	}
	id := r.PathValue("id")
	p, ok := s.agents.Get(id)
	if !ok {
		reason(w, http.StatusConflict, "not-paired")
		return
	}
	a, ok := s.reg.Get(id)
	if !ok {
		reason(w, http.StatusServiceUnavailable, "offline")
		return
	}
	target, err := url.Parse(agentURL(a))
	if err != nil {
		reason(w, http.StatusServiceUnavailable, "offline")
		return
	}
	// Chemin brut (encore encodé) : un %2F dans un paramètre reste un %2F.
	rest := strings.TrimPrefix(r.URL.EscapedPath(), "/api/pc/"+url.PathEscape(id))
	rp := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.Out.URL.Scheme, pr.Out.URL.Host = target.Scheme, target.Host
			pr.Out.URL.RawPath = "/api" + rest
			pr.Out.URL.Path, _ = url.PathUnescape(pr.Out.URL.RawPath)
			pr.Out.Host = target.Host
			pr.Out.Header.Del("Cookie") // la clé de la tablette ne sort pas du hub
			pr.Out.Header.Set("Authorization", "Bearer "+p.Token)
		},
		Transport: s.transport(p.Fingerprint),
		ModifyResponse: func(resp *http.Response) error {
			if resp.StatusCode == http.StatusUnauthorized {
				// Le PC ne reconnaît plus le hub (devices.json effacé) : il faudra le réappairer.
				log.Printf("%s a révoqué le hub : à réappairer", a.Name)
				if err := s.agents.Forget(id); err != nil {
					log.Printf("oubli de %s : %v", a.Name, err)
				}
			}
			resp.Header.Del("Set-Cookie")
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if errors.Is(err, agents.ErrWrongPC) {
				log.Printf("%s : %v", a.Name, err)
				reason(w, http.StatusBadGateway, "wrong-pc")
				return
			}
			if r.Context().Err() == nil {
				log.Printf("relais vers %s : %v", a.Name, err)
			}
			reason(w, http.StatusBadGateway, "offline")
		},
	}
	rp.ServeHTTP(w, r)
}

// transport renvoie le client HTTPS d'un PC, réutilisé d'une requête à l'autre
// (connexions gardées ouvertes).
func (s *Server) transport(fingerprint string) http.RoundTripper {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.transports[fingerprint]
	if !ok {
		t = agents.Transport(fingerprint, nil)
		s.transports[fingerprint] = t
	}
	return t
}
