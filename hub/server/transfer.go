package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// Envoi d'un fichier d'un PC à un autre : le hub lit le fichier sur l'agent
// source et le pousse au fur et à mesure vers l'agent cible, sans passer par la
// tablette. La réponse est un flux de lignes JSON pour suivre l'avancement :
//
//	{"sent":1048576,"total":5242880}
//	{"done":true,"name":"photo (1).jpg"}   ou   {"error":"target-denied"}

type transferReq struct {
	From string `json:"from"`
	To   string `json:"to"`
	Path string `json:"path"`
	Dir  string `json:"dir"` // dossier du PC cible ; vide : ses Téléchargements
}

type transferEvent struct {
	Sent  int64  `json:"sent,omitempty"`
	Total int64  `json:"total,omitempty"`
	Done  bool   `json:"done,omitempty"`
	Name  string `json:"name,omitempty"`
	Error string `json:"error,omitempty"`
}

// Fréquence des nouvelles d'avancement envoyées à la tablette.
const progressEvery = 250 * time.Millisecond

// transferError porte la raison lue par l'appli (« not-found », « offline »…).
type transferError string

func (e transferError) Error() string { return string(e) }

func (s *Server) handleTransfer(w http.ResponseWriter, r *http.Request) {
	if !s.tabletPaired(r) {
		reason(w, http.StatusUnauthorized, "tablet")
		return
	}
	var req transferReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Path == "" || req.From == "" || req.To == "" {
		reason(w, http.StatusBadRequest, "bad-request")
		return
	}
	if req.From == req.To {
		reason(w, http.StatusBadRequest, "same-pc")
		return
	}
	src, ok := s.endpoint(req.From)
	if !ok {
		reason(w, http.StatusServiceUnavailable, "offline")
		return
	}
	dst, ok := s.endpoint(req.To)
	if !ok {
		reason(w, http.StatusServiceUnavailable, "target-offline")
		return
	}

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, _ := w.(http.Flusher)
	enc := json.NewEncoder(w)
	send := func(e transferEvent) {
		if r.Context().Err() != nil {
			return // la tablette est partie : l'envoi continue sans elle
		}
		enc.Encode(e)
		if flusher != nil {
			flusher.Flush()
		}
	}

	// L'envoi ne dépend pas de la tablette : il va au bout même si elle se met en veille.
	var sent, total atomic.Int64
	result := make(chan transferEvent, 1)
	go func() {
		name, err := s.transfer(context.WithoutCancel(r.Context()), src, dst, req.Path, req.Dir, &sent, &total)
		if err != nil {
			var te transferError
			if !errors.As(err, &te) {
				te = "failed"
			}
			log.Printf("envoi de %s de %s vers %s : %v", req.Path, src.name, dst.name, err)
			result <- transferEvent{Error: string(te)}
			return
		}
		log.Printf("envoi de %s de %s vers %s : %s", req.Path, src.name, dst.name, name)
		result <- transferEvent{Done: true, Name: name}
	}()

	tick := time.NewTicker(progressEvery)
	defer tick.Stop()
	for {
		select {
		case e := <-result:
			send(e)
			return
		case <-tick.C:
			if t := total.Load(); t > 0 {
				send(transferEvent{Sent: sent.Load(), Total: t})
			}
		}
	}
}

// endpoint : de quoi parler à un PC appairé et en ligne.
type endpoint struct {
	name      string
	base      string
	token     string
	transport http.RoundTripper
}

func (s *Server) endpoint(id string) (endpoint, bool) {
	p, ok := s.agents.Get(id)
	if !ok {
		return endpoint{}, false
	}
	a, ok := s.reg.Get(id)
	if !ok {
		return endpoint{}, false
	}
	return endpoint{name: a.Name, base: agentURL(a), token: p.Token, transport: s.transport(p.Fingerprint)}, true
}

func (e endpoint) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+e.token)
	return (&http.Client{Transport: e.transport}).Do(req)
}

// transfer copie path de src vers le dossier dir de dst (vide : Téléchargements)
// et renvoie le nom sous lequel il y a été enregistré.
func (s *Server) transfer(ctx context.Context, src, dst endpoint, path, dir string, sent, total *atomic.Int64) (string, error) {
	get, err := http.NewRequestWithContext(ctx, http.MethodGet, src.base+"/api/files/download?path="+url.QueryEscape(path), nil)
	if err != nil {
		return "", err
	}
	in, err := src.do(get)
	if err != nil {
		return "", errors.Join(transferError("offline"), err)
	}
	defer in.Body.Close()
	switch in.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return "", transferError("not-found")
	case http.StatusForbidden:
		return "", transferError("denied")
	default:
		return "", errors.Join(transferError("failed"), agentStatus(in))
	}
	total.Store(max(in.ContentLength, 0))

	put, err := http.NewRequestWithContext(ctx, http.MethodPost,
		dst.base+"/api/files/receive?"+url.Values{"name": {baseName(path)}, "dir": {dir}}.Encode(), &counter{r: in.Body, n: sent})
	if err != nil {
		return "", err
	}
	put.ContentLength = in.ContentLength
	put.Header.Set("Content-Type", "application/octet-stream")
	out, err := dst.do(put)
	if err != nil {
		if ctx.Err() == nil && sent.Load() < in.ContentLength {
			// La lecture a pu échouer côté source comme l'écriture côté cible.
			return "", errors.Join(transferError("failed"), err)
		}
		return "", errors.Join(transferError("target-offline"), err)
	}
	defer out.Body.Close()
	switch out.StatusCode {
	case http.StatusOK:
	case http.StatusForbidden:
		return "", transferError("target-denied")
	case http.StatusNotFound, http.StatusBadRequest:
		return "", transferError("target-folder") // dossier supprimé ou déplacé depuis
	default:
		return "", errors.Join(transferError("target-failed"), agentStatus(out))
	}
	var saved struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	if err := json.NewDecoder(out.Body).Decode(&saved); err != nil {
		return "", errors.Join(transferError("target-failed"), err)
	}
	// Un agent d'avant le choix du dossier ignore dir et enregistre dans
	// Téléchargements : on le signale au lieu de dire « envoyé dans … ».
	if dir != "" && !sameDir(parentDir(saved.Path), dir) {
		return "", errors.Join(transferError("target-outdated"), errors.New("enregistré dans "+saved.Path))
	}
	return saved.Name, nil
}

// parentDir : dossier d'un chemin Windows ou Linux, quel que soit l'OS du hub.
func parentDir(path string) string {
	return strings.TrimSuffix(path, baseName(path))
}

// sameDir compare deux dossiers sans tenir compte du séparateur final, ni de la
// casse pour un chemin Windows.
func sameDir(a, b string) bool {
	a, b = strings.TrimRight(a, `/\`), strings.TrimRight(b, `/\`)
	if strings.Contains(a, `\`) {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func agentStatus(resp *http.Response) error {
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return errors.New(strconv.Itoa(resp.StatusCode) + " " + string(b))
}

// baseName : dernier élément d'un chemin Windows ou Linux, quel que soit l'OS du hub.
func baseName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}

// counter compte les octets lus, pour l'avancement.
type counter struct {
	r io.Reader
	n *atomic.Int64
}

func (c *counter) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n.Add(int64(n))
	return n, err
}
