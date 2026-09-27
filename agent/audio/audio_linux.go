package audio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Sous Linux on passe par pactl, fourni avec PulseAudio et avec PipeWire
// (pipewire-pulse) : il marche avec les deux serveurs de son courants.
// Sa sortie JSON (-f json) existe depuis PulseAudio 16 / PipeWire 0.3.50.

type paVolume map[string]struct {
	Value int `json:"value"` // 65536 = 100 %
}

type paDevice struct {
	Index       int               `json:"index"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Mute        bool              `json:"mute"`
	Volume      paVolume          `json:"volume"`
	Properties  map[string]string `json:"properties"`
}

type paStream struct {
	Index      int               `json:"index"`
	Mute       bool              `json:"mute"`
	Volume     paVolume          `json:"volume"`
	Properties map[string]string `json:"properties"`
}

type paInfo struct {
	DefaultSink   string `json:"default_sink_name"`
	DefaultSource string `json:"default_source_name"`
}

func pactl(ctx context.Context, args ...string) ([]byte, error) {
	if _, err := exec.LookPath("pactl"); err != nil {
		return nil, errors.New("pactl introuvable : installez pulseaudio-utils (ou pipewire-pulse)")
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "pactl", args...).Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return nil, fmt.Errorf("pactl %s : %s", args[0], strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, fmt.Errorf("pactl %s : %w", args[0], err)
	}
	return out, nil
}

func pactlJSON(ctx context.Context, v any, args ...string) error {
	out, err := pactl(ctx, append([]string{"-f", "json"}, args...)...)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(out, v); err != nil {
		return fmt.Errorf("pactl %s : réponse illisible (pactl trop ancien ?)", strings.Join(args, " "))
	}
	return nil
}

func read(ctx context.Context) (State, error) {
	var info paInfo
	var sinks, sources []paDevice
	var inputs []paStream
	if err := pactlJSON(ctx, &info, "info"); err != nil {
		return State{}, err
	}
	if err := pactlJSON(ctx, &sinks, "list", "sinks"); err != nil {
		return State{}, err
	}
	if err := pactlJSON(ctx, &sources, "list", "sources"); err != nil {
		return State{}, err
	}
	if err := pactlJSON(ctx, &inputs, "list", "sink-inputs"); err != nil {
		return State{}, err
	}
	return buildState(info, sinks, sources, inputs), nil
}

func buildState(info paInfo, sinks, sources []paDevice, inputs []paStream) State {
	var s State
	for _, d := range sinks {
		isDefault := d.Name == info.DefaultSink
		s.Outputs = append(s.Outputs, Device{ID: d.Name, Name: d.label(), Default: isDefault})
		if isDefault {
			s.Master = &Level{Volume: d.Volume.percent(), Muted: d.Mute}
		}
	}
	for _, d := range sources {
		if d.Name == info.DefaultSource && !d.isMonitor() {
			s.Mic = &Level{Volume: d.Volume.percent(), Muted: d.Mute}
		}
	}

	byID := map[string]int{}
	for _, in := range inputs {
		id, name := in.app()
		if id == "" {
			continue
		}
		lvl := Level{Volume: in.Volume.percent(), Muted: in.Mute}
		if i, ok := byID[id]; ok {
			a := &s.Apps[i]
			a.Volume = max(a.Volume, lvl.Volume)
			a.Muted = a.Muted && lvl.Muted // coupée seulement si tous ses flux le sont
			continue
		}
		byID[id] = len(s.Apps)
		s.Apps = append(s.Apps, App{ID: id, Name: name, Level: lvl, exe: in.exe()})
	}
	return s
}

func (d paDevice) label() string {
	if d.Description != "" {
		return d.Description
	}
	return d.Name
}

func (d paDevice) isMonitor() bool {
	return d.Properties["device.class"] == "monitor" || strings.HasSuffix(d.Name, ".monitor")
}

// app identifie le programme d'un flux : son exécutable, à défaut son nom.
func (s paStream) app() (id, name string) {
	bin := s.Properties["application.process.binary"]
	if bin == "" {
		bin = s.Properties["application.name"]
	}
	if bin == "" {
		return "", ""
	}
	return strings.ToLower(bin), appName(bin)
}

func (s paStream) exe() string {
	if pid := s.Properties["application.process.id"]; pid != "" {
		if p, err := os.Readlink(filepath.Join("/proc", pid, "exe")); err == nil {
			return p
		}
	}
	return s.Properties["application.process.binary"]
}

// percent renvoie la moyenne des canaux, bornée à 100 (pactl autorise jusqu'à 150 %).
func (v paVolume) percent() int {
	if len(v) == 0 {
		return 0
	}
	sum := 0
	for _, c := range v {
		sum += c.Value
	}
	p := int(math.Round(float64(sum) / float64(len(v)) / 65536 * 100))
	return min(p, 100)
}

func setVolume(ctx context.Context, t target, volume int) error {
	v := strconv.Itoa(volume) + "%"
	switch t.kind {
	case "master":
		_, err := pactl(ctx, "set-sink-volume", "@DEFAULT_SINK@", v)
		return err
	case "mic":
		_, err := pactl(ctx, "set-source-volume", "@DEFAULT_SOURCE@", v)
		return err
	}
	return forApp(ctx, t.app, "set-sink-input-volume", v)
}

func setMute(ctx context.Context, t target, muted bool) error {
	m := "0"
	if muted {
		m = "1"
	}
	switch t.kind {
	case "master":
		_, err := pactl(ctx, "set-sink-mute", "@DEFAULT_SINK@", m)
		return err
	case "mic":
		_, err := pactl(ctx, "set-source-mute", "@DEFAULT_SOURCE@", m)
		return err
	}
	return forApp(ctx, t.app, "set-sink-input-mute", m)
}

// forApp applique la commande à tous les flux de l'application.
func forApp(ctx context.Context, id, cmd, value string) error {
	var inputs []paStream
	if err := pactlJSON(ctx, &inputs, "list", "sink-inputs"); err != nil {
		return err
	}
	found := false
	for _, in := range inputs {
		if appID, _ := in.app(); appID != id {
			continue
		}
		found = true
		if _, err := pactl(ctx, cmd, strconv.Itoa(in.Index), value); err != nil {
			return err
		}
	}
	if !found {
		return ErrNotFound
	}
	return nil
}

func setOutput(ctx context.Context, id string) error {
	var sinks []paDevice
	if err := pactlJSON(ctx, &sinks, "list", "sinks"); err != nil {
		return err
	}
	found := false
	for _, d := range sinks {
		found = found || d.Name == id
	}
	if !found {
		return ErrNotFound
	}
	if _, err := pactl(ctx, "set-default-sink", id); err != nil {
		return err
	}
	// PulseAudio laisse les sons en cours sur l'ancienne sortie : on les déplace.
	var inputs []paStream
	if err := pactlJSON(ctx, &inputs, "list", "sink-inputs"); err == nil {
		for _, in := range inputs {
			pactl(ctx, "move-sink-input", strconv.Itoa(in.Index), id)
		}
	}
	return nil
}
