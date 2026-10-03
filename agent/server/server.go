// Package server expose l'API HTTP et sert l'appli tablette embarquée.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mickael-pezzoni/deckpad/agent/audio"
	"github.com/mickael-pezzoni/deckpad/agent/auth"
	"github.com/mickael-pezzoni/deckpad/agent/clipboard"
	"github.com/mickael-pezzoni/deckpad/agent/files"
	"github.com/mickael-pezzoni/deckpad/agent/live"
	"github.com/mickael-pezzoni/deckpad/agent/media"
	"github.com/mickael-pezzoni/deckpad/agent/network"
	"github.com/mickael-pezzoni/deckpad/agent/notify"
	"github.com/mickael-pezzoni/deckpad/agent/process"
	"github.com/mickael-pezzoni/deckpad/agent/shortcuts"
	"github.com/mickael-pezzoni/deckpad/agent/stats"
	"github.com/mickael-pezzoni/deckpad/agent/sysinfo"
	"github.com/mickael-pezzoni/deckpad/agent/system"
)

// New construit le routeur de l'API, appelée par le hub (l'appli est servie par le hub).
// Seul le hub appairé dans store y accède.
func New(store *auth.Store, keys *shortcuts.Store, favs *files.Favorites) http.Handler {
	procs := process.NewLister()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/pair/start", handlePairStart(store))
	mux.HandleFunc("POST /api/pair/confirm", handlePairConfirm(store))
	mux.HandleFunc("GET /api/pair/code", handlePairCode(store))
	mux.HandleFunc("GET /pair-code", localOnly(handlePairWindow))
	mux.HandleFunc("GET /api/info", handleInfo)
	mux.Handle("GET /api/stats/stream", stream(live.NewHub(time.Second, stats.Collect)))
	mux.Handle("GET /api/processes/stream", stream(live.NewHub(2*time.Second, procs.Apps)))
	mux.HandleFunc("POST /api/processes/kill", handleKill(procs))
	mux.HandleFunc("GET /api/processes/icon", handleIcon(procs))
	mux.Handle("GET /api/network/stream", stream(live.NewHub(time.Second, network.NewMonitor().Collect)))
	mux.HandleFunc("GET /api/network/public-ip", handlePublicIP)
	mux.HandleFunc("POST /api/system/{action}", handleSystem)
	mux.HandleFunc("GET /api/files/drives", handleDrives)
	mux.HandleFunc("GET /api/files/list", handleList)
	mux.HandleFunc("GET /api/files/recent", handleRecent)
	mux.HandleFunc("GET /api/files/download", handleDownload)
	mux.HandleFunc("POST /api/files/receive", handleReceive)
	mux.HandleFunc("POST /api/files/open", handleFileAction(files.OpenOnPC))
	mux.HandleFunc("POST /api/files/reveal", handleFileAction(files.Reveal))
	mux.HandleFunc("GET /api/files/favorites", handleFavorites(favs))
	mux.HandleFunc("POST /api/files/favorites", handleFavoriteSet(favs))
	mux.Handle("GET /api/audio/stream", stream(live.NewHub(time.Second, audio.Collect)))
	mux.HandleFunc("POST /api/audio/volume", handleAudioVolume)
	mux.HandleFunc("POST /api/audio/mute", handleAudioMute)
	mux.HandleFunc("POST /api/audio/output", handleAudioOutput)
	mux.HandleFunc("GET /api/audio/icon", handleAudioIcon)
	mux.Handle("GET /api/media/stream", stream(live.NewHub(time.Second, media.Collect)))
	mux.HandleFunc("POST /api/media/{action}", handleMediaControl)
	mux.HandleFunc("GET /api/media/cover", handleMediaCover)
	mux.HandleFunc("GET /api/shortcuts", handleShortcuts(keys))
	mux.HandleFunc("PUT /api/shortcuts", handleShortcutsSave(keys))
	mux.HandleFunc("POST /api/shortcuts/{id}/run", handleShortcutRun(keys))
	mux.Handle("GET /api/clipboard/stream", stream(live.NewHub(time.Second, clipboard.Collect)))
	mux.HandleFunc("GET /api/clipboard/image", handleClipboardImage)
	mux.HandleFunc("POST /api/clipboard/send", handleClipboardSend)
	return requireToken(store, mux)
}

func handleInfo(w http.ResponseWriter, r *http.Request) {
	info, err := sysinfo.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, info)
}

func handlePublicIP(w http.ResponseWriter, r *http.Request) {
	ip, err := network.PublicIP(r.Context())
	if err != nil {
		http.Error(w, "IP publique indisponible", http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]string{"ip": ip})
}

func handleIcon(procs *process.Lister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		icon, err := procs.Icon(r.Context(), r.URL.Query().Get("name"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", icon.ContentType)
		w.Header().Set("Cache-Control", "max-age=86400")
		w.Write(icon.Data)
	}
}

func handleDrives(w http.ResponseWriter, r *http.Request) {
	drives, err := files.Drives(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, drives)
}

func handleRecent(w http.ResponseWriter, r *http.Request) {
	recents, err := files.Recents(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, recents)
}

func handleList(w http.ResponseWriter, r *http.Request) {
	l, err := files.List(r.Context(), r.URL.Query().Get("path"))
	if err != nil {
		filesError(w, err)
		return
	}
	writeJSON(w, l)
}

// filesError traduit les erreurs du paquet files en codes HTTP.
func filesError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, files.ErrOutside), errors.Is(err, files.ErrForbidden):
		http.Error(w, err.Error(), http.StatusForbidden)
	case errors.Is(err, files.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, files.ErrNotFile), errors.Is(err, files.ErrTooMany), errors.Is(err, files.ErrBadName), errors.Is(err, files.ErrNotDir):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleDownload envoie le fichier à la tablette, qui l'enregistre.
func handleDownload(w http.ResponseWriter, r *http.Request) {
	path, info, err := files.File(r.Context(), r.URL.Query().Get("path"))
	if err != nil {
		filesError(w, err)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		filesError(w, err)
		return
	}
	defer f.Close()
	name := filepath.Base(path)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	log.Printf("téléchargement : %s", path)
	http.ServeContent(w, r, name, info.ModTime(), f)
}

// handleReceive enregistre un fichier envoyé depuis un autre PC (via le hub)
// dans le dossier demandé (par défaut Téléchargements), et renvoie son nom final.
func handleReceive(w http.ResponseWriter, r *http.Request) {
	path, err := files.Receive(r.Context(), r.URL.Query().Get("dir"), r.URL.Query().Get("name"), r.Body)
	if err != nil {
		log.Printf("réception : %v", err)
		filesError(w, err)
		return
	}
	log.Printf("fichier reçu : %s", path)
	notify.Send("Fichier reçu", filepath.Base(path)+" · dans "+filepath.Base(filepath.Dir(path)))
	writeJSON(w, map[string]string{"name": filepath.Base(path), "path": path})
}

// handleFileAction : ouvrir le fichier sur le PC ou le montrer dans l'Explorateur.
func handleFileAction(action func(context.Context, string) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "requête invalide", http.StatusBadRequest)
			return
		}
		if err := action(r.Context(), req.Path); err != nil {
			filesError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleFavorites(favs *files.Favorites) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := favs.List(r.Context())
		if err != nil {
			filesError(w, err)
			return
		}
		writeJSON(w, list)
	}
}

// handleFavoriteSet ajoute ou retire un favori, et renvoie la liste à jour.
func handleFavoriteSet(favs *files.Favorites) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Path     string `json:"path"`
			Favorite bool   `json:"favorite"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "requête invalide", http.StatusBadRequest)
			return
		}
		if err := favs.Set(r.Context(), req.Path, req.Favorite); err != nil {
			filesError(w, err)
			return
		}
		handleFavorites(favs)(w, r)
	}
}

func handleAudioVolume(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Target string `json:"target"`
		Volume int    `json:"volume"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "requête invalide", http.StatusBadRequest)
		return
	}
	audioResult(w, audio.SetVolume(r.Context(), req.Target, req.Volume))
}

func handleAudioMute(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Target string `json:"target"`
		Muted  bool   `json:"muted"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "requête invalide", http.StatusBadRequest)
		return
	}
	audioResult(w, audio.SetMute(r.Context(), req.Target, req.Muted))
}

func handleAudioOutput(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "requête invalide", http.StatusBadRequest)
		return
	}
	err := audio.SetOutput(r.Context(), req.ID)
	if err == nil {
		log.Printf("sortie audio : %s", req.ID)
	}
	audioResult(w, err)
}

func audioResult(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, audio.ErrBadTarget):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, audio.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleAudioIcon(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("app")
	exe, ok := audio.AppExe(id)
	if !ok {
		http.Error(w, audio.ErrNotFound.Error(), http.StatusNotFound)
		return
	}
	icon, err := process.IconOf(exe, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", icon.ContentType)
	w.Header().Set("Cache-Control", "max-age=86400")
	w.Write(icon.Data)
}

func handleMediaControl(w http.ResponseWriter, r *http.Request) {
	err := media.Control(r.Context(), media.Action(r.PathValue("action")))
	switch {
	case errors.Is(err, media.ErrBadAction):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, media.ErrNoPlayer):
		http.Error(w, err.Error(), http.StatusNotFound)
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleMediaCover sert la pochette du morceau en cours ; l'adresse change avec
// le morceau (?v=…), elle peut donc rester en cache.
func handleMediaCover(w http.ResponseWriter, r *http.Request) {
	data, contentType, err := media.Cover(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "max-age=86400")
	w.Write(data)
}

type shortcutsReply struct {
	Shortcuts []shortcuts.Shortcut   `json:"shortcuts"`
	Keys      shortcuts.Availability `json:"keys"`
	Capture   shortcuts.Availability `json:"capture"`
}

func newShortcutsReply(list []shortcuts.Shortcut) shortcutsReply {
	return shortcutsReply{list, shortcuts.KeysAvailable(), shortcuts.CaptureAvailable()}
}

func handleShortcuts(keys *shortcuts.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, newShortcutsReply(keys.List()))
	}
}

// handleShortcutsSave remplace toute la liste (ajout, modification, suppression).
func handleShortcutsSave(keys *shortcuts.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var list []shortcuts.Shortcut
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&list); err != nil {
			http.Error(w, "requête invalide", http.StatusBadRequest)
			return
		}
		saved, err := keys.Replace(list)
		switch {
		case errors.Is(err, shortcuts.ErrInvalid):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case err != nil:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		default:
			writeJSON(w, newShortcutsReply(saved))
		}
	}
}

func handleShortcutRun(keys *shortcuts.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		err := keys.Run(id)
		switch {
		case errors.Is(err, shortcuts.ErrNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case err != nil:
			log.Printf("raccourci %s : %v", id, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

// handleClipboardImage sert l'image du presse-papiers ; l'adresse change avec
// l'image (?v=…), elle peut donc rester en cache.
func handleClipboardImage(w http.ResponseWriter, r *http.Request) {
	data, err := clipboard.Image()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "max-age=86400")
	if r.URL.Query().Has("download") {
		name := "presse-papiers-" + time.Now().Format("2006-01-02-150405") + ".png"
		w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	}
	w.Write(data)
}

// handleClipboardSend reçoit un texte de la tablette : copié dans le
// presse-papiers, ouvert dans le navigateur ou tapé dans la fenêtre active.
func handleClipboardSend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"` // copy, open ou type
		Text   string `json:"text"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, clipboard.MaxText+1024)).Decode(&req); err != nil {
		http.Error(w, "requête invalide", http.StatusBadRequest)
		return
	}
	var err error
	switch req.Action {
	case "copy":
		err = clipboard.SetText(r.Context(), req.Text)
	case "open":
		err = shortcuts.OpenURL(req.Text)
	case "type":
		err = shortcuts.TypeText(req.Text)
	default:
		http.Error(w, "action inconnue", http.StatusBadRequest)
		return
	}
	switch {
	case errors.Is(err, clipboard.ErrEmpty), errors.Is(err, clipboard.ErrTooLong),
		errors.Is(err, shortcuts.ErrBadURL), errors.Is(err, shortcuts.ErrLongText):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case err != nil:
		log.Printf("presse-papiers (%s) : %v", req.Action, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	default:
		if req.Action == "copy" {
			notify.Send("Texte copié depuis la tablette", notify.Excerpt(req.Text, 80))
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleSystem(w http.ResponseWriter, r *http.Request) {
	action := system.Action(r.PathValue("action"))
	if err := system.Run(action); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("action système : %s", action)
	w.WriteHeader(http.StatusAccepted)
}

func handleKill(procs *process.Lister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
			http.Error(w, "nom manquant", http.StatusBadRequest)
			return
		}
		n, err := procs.Kill(r.Context(), req.Name)
		switch {
		case errors.Is(err, process.ErrNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case err != nil:
			http.Error(w, err.Error(), http.StatusForbidden) // souvent : droits admin requis
		default:
			log.Printf("processus %q fermé (%d)", req.Name, n)
			notify.Send("Application fermée depuis la tablette", req.Name)
			writeJSON(w, map[string]int{"killed": n})
		}
	}
}

// stream pousse chaque mesure du hub à la tablette (Server-Sent Events).
func stream[T any](hub *live.Hub[T]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		rc := http.NewResponseController(w)

		ch, unsubscribe := hub.Subscribe()
		defer unsubscribe()
		for {
			select {
			case <-r.Context().Done():
				return
			case v := <-ch:
				data, err := json.Marshal(v)
				if err != nil {
					log.Printf("JSON : %v", err)
					continue
				}
				if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
					return
				}
				if err := rc.Flush(); err != nil {
					return
				}
			}
		}
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("écriture JSON : %v", err)
	}
}
