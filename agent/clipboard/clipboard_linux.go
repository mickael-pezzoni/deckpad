package clipboard

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
	"syscall"
	"time"
)

// Sous Linux, on passe par les outils du bureau : xclip (ou xsel) sous X11,
// wl-clipboard (wl-copy / wl-paste) sous Wayland.
//
// Pour lire, xclip est préféré même sous Wayland quand XWayland tourne : GNOME et
// KDE y recopient le presse-papiers, et wl-paste, sous GNOME, doit ouvrir une
// fenêtre invisible à chaque lecture (lue chaque seconde, l'écran clignoterait).

var errNoTool = errors.New("installe xclip (ou wl-clipboard sous Wayland) pour lire le presse-papiers")

func wayland() bool {
	return os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("XDG_SESSION_TYPE") == "wayland"
}

func x11() bool { return os.Getenv("DISPLAY") != "" }

func has(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// reader : outil de lecture du presse-papiers, choisi selon la session.
type reader struct {
	types func(ctx context.Context) ([]string, error) // nil : texte seulement
	text  func(ctx context.Context, types []string) ([]byte, error)
	image func(ctx context.Context) ([]byte, error)
}

func pickReader() (reader, error) {
	switch {
	case x11() && has("xclip"):
		xclip := func(ctx context.Context, target string, limit int) ([]byte, error) {
			return output(ctx, limit, "xclip", "-selection", "clipboard", "-o", "-t", target)
		}
		return reader{
			types: func(ctx context.Context) ([]string, error) {
				b, err := xclip(ctx, "TARGETS", 64<<10)
				return strings.Fields(string(b)), err
			},
			text: func(ctx context.Context, types []string) ([]byte, error) {
				for _, t := range []string{"UTF8_STRING", "text/plain;charset=utf-8", "STRING", "TEXT"} {
					if slices.Contains(types, t) {
						return xclip(ctx, t, MaxText)
					}
				}
				return nil, nil
			},
			image: func(ctx context.Context) ([]byte, error) { return xclip(ctx, "image/png", maxImage) },
		}, nil
	case wayland() && has("wl-paste"):
		return reader{
			types: func(ctx context.Context) ([]string, error) {
				b, err := output(ctx, 64<<10, "wl-paste", "--list-types")
				return strings.Split(strings.TrimSpace(string(b)), "\n"), err
			},
			text: func(ctx context.Context, types []string) ([]byte, error) {
				if !slices.ContainsFunc(types, isText) {
					return nil, nil
				}
				return output(ctx, MaxText, "wl-paste", "--no-newline", "--type", "text")
			},
			image: func(ctx context.Context) ([]byte, error) {
				return output(ctx, maxImage, "wl-paste", "--type", "image/png")
			},
		}, nil
	case x11() && has("xsel"):
		return reader{text: func(ctx context.Context, _ []string) ([]byte, error) {
			return output(ctx, MaxText, "xsel", "--clipboard", "--output")
		}}, nil
	case !x11() && !wayland():
		return reader{}, errors.New("aucune session graphique (lance l'agent depuis ta session)")
	}
	return reader{}, errNoTool
}

func isText(t string) bool {
	return strings.HasPrefix(t, "text/plain") || t == "UTF8_STRING" || t == "STRING" || t == "TEXT"
}

func read(ctx context.Context) (content, error) {
	r, err := pickReader()
	if err != nil {
		return content{}, err
	}
	var types []string
	if r.types != nil {
		if types, err = r.types(ctx); err != nil {
			return content{}, nil // presse-papiers vide : aucun programme ne le possède
		}
	}
	var c content
	if b, err := r.text(ctx, types); err == nil {
		c.text = string(b)
	}
	if r.image != nil && slices.Contains(types, "image/png") {
		if b, err := r.image(ctx); err == nil {
			c.image = b
		}
	}
	return c, nil
}

// output lance un outil et renvoie sa sortie (limit octets au plus).
func output(ctx context.Context, limit int, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	var out bytes.Buffer
	cmd.Stdout = &limitWriter{w: &out, n: limit}
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s : %w", name, err)
	}
	return out.Bytes(), nil
}

// limitWriter garde les n premiers octets et ignore la suite (l'outil n'est
// pas interrompu en pleine écriture).
type limitWriter struct {
	w io.Writer
	n int
}

func (l *limitWriter) Write(p []byte) (int, error) {
	if l.n > 0 {
		k := min(len(p), l.n)
		l.w.Write(p[:k])
		l.n -= k
	}
	return len(p), nil
}

func write(ctx context.Context, text string) error {
	var name string
	var args []string
	switch {
	case wayland() && has("wl-copy"):
		name, args = "wl-copy", nil
	case x11() && has("xclip"):
		name, args = "xclip", []string{"-selection", "clipboard", "-in"}
	case x11() && has("xsel"):
		name, args = "xsel", []string{"--clipboard", "--input"}
	case !x11() && !wayland():
		return errors.New("aucune session graphique (lance l'agent depuis ta session)")
	default:
		return errors.New("installe xclip (ou wl-clipboard sous Wayland) pour copier dans le presse-papiers")
	}
	// Ces outils restent en arrière-plan pour fournir le texte aux applis : on ne
	// garde aucun tuyau ouvert vers eux (sinon Wait attendrait leur fin).
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = strings.NewReader(text)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s : %w", name, err)
	}
	return nil
}
