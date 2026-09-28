# deckpad

Control your PC from a tablet while you play: themed pages you swipe through, with big tiles.

- `agent/`: Go program running on the PC (API + embedded tablet app), a single `.exe`.
- `web/`: tablet app (React + Vite + Swiper), installable as a PWA. Available in English and French.

## Preview

![PC info](docs/screenshots/info.png)

| | |
|---|---|
| ![Stats](docs/screenshots/stats.png) Stats | ![Processes](docs/screenshots/processes.png) Processes |
| ![Network](docs/screenshots/network.png) Network | ![Audio](docs/screenshots/audio.png) Audio |
| ![Media](docs/screenshots/media.png) Media | ![Shortcuts](docs/screenshots/shortcuts.png) Shortcuts |
| ![Files](docs/screenshots/files.png) Files | ![Clipboard](docs/screenshots/clipboard.png) Clipboard |
| ![System](docs/screenshots/system.png) System | ![Light theme](docs/screenshots/info-light.png) Light theme |
| ![Settings](docs/screenshots/settings.png) Settings | ![In French](docs/screenshots/processes-french.png) In French |

On phones too:

<p>
  <img src="docs/screenshots/stats-phone.png" alt="Stats on a phone" width="240">
  <img src="docs/screenshots/network-phone.png" alt="Network on a phone" width="240">
</p>

## Usage

1. Run `deckpad.exe` on the PC. It shows the address to open (e.g. `http://192.168.1.20:8420`).
2. Open that address on the tablet, then "Add to Home screen".

### Install the app on the tablet (PWA)

Browsers only install a real app (full screen, icon, cache) over HTTPS. So deckpad also serves the app over HTTPS on port 8421, with its own certificate.

Right after pairing, the tablet offers to switch to a secure connection, in 3 steps (only once):

1. "Download the certificate".
2. Android: in Settings, search for "CA certificate" and pick `deckpad-ca.crt`. iPad: Settings › Profile Downloaded › Install, then Settings › General › About › Certificate Trust Settings, turn on deckpad.
3. "Continue securely": the tablet switches to HTTPS without pairing again. All that's left is installing the app (⋮ menu → "Install app" in Chrome).

Good to know:

- On first launch, deckpad creates its own small certificate authority in `%APPDATA%\deckpad` (Windows) or `~/.config/deckpad` (Linux): `ca.crt` and `ca.key`. The key never leaves the PC. It can only sign local network addresses, so it would be useless to impersonate another website.
- If the PC's address changes, the certificate is regenerated automatically, with nothing to reinstall on the tablet.
- Android then shows "network may be monitored": this is normal after installing a certificate.
- `-https-addr ""` disables HTTPS. In development mode, HTTPS serves the built app: run `npm run build` before testing this step.

Without a certificate, on Android, you can also enable `chrome://flags/#unsafely-treat-insecure-origin-as-secure` with the PC's HTTP address, then install the app.

### Pair a tablet

As long as no tablet is paired, the PC opens a small window at launch with a QR code of the address to open. The tablet then shows a keypad (with a language picker, English by default) and the window a 6-digit code (also printed in the console). Type that code on the tablet: it receives a key and won't have to do it again.

- The code expires after 5 minutes and locks after 5 wrong attempts.
- Each device has its own key. The PC only keeps its fingerprint, in `%APPDATA%\deckpad\devices.json` (Windows) or `~/.config/deckpad/devices.json` (Linux). Deleting this file unpairs everything.
- The window opens with Edge or Chrome (Chromium on Linux), otherwise in the default browser.

## Build the exe

Requirements: [Go](https://go.dev/dl/) and [Node.js](https://nodejs.org/).

```
.\build.ps1      # Windows
./build.sh       # macOS / Linux
```

Output: `bin/deckpad.exe`.

## Develop

Two terminals:

```
cd agent && go run .        # API on :8420
cd web && npm install && npm run dev
```

Open the address shown by Vite (on the PC or the tablet). The app reloads on every save and `/api` calls are proxied to the agent.

App texts: `web/src/i18n/en.ts` and `web/src/i18n/fr.ts` (react-i18next). A text added in one language must also be added in the other: the build fails otherwise.

## Pages

| Page | Status |
|---|---|
| PC info | ✅ |
| Stats | ✅ CPU, RAM, GPU of any brand (FPS and AMD GPU temperature on Windows to come) |
| Processes | ✅ user applications, sort by CPU/RAM, close with a long press |
| Files | ✅ drives and USB sticks, folder browsing (read-only) |
| Network | ✅ live throughput, latency, local and public IP, connection type |
| Audio | ✅ master and per-app volume, microphone, output choice (headset, speakers…). On Linux: `pactl` required (ships with PulseAudio / PipeWire) |
| Media | ✅ title, artist, cover, play/pause, next/previous (Spotify, YouTube in the browser, VLC…). Windows 10 1809+; on Linux, any MPRIS-compatible player |
| Shortcuts | ✅ tiles configured from the tablet (long press to edit): key combination, program, folder or web address, screen capture copied to the clipboard. Stored in `%APPDATA%\deckpad\shortcuts.json` (Windows) or `~/.config/deckpad/shortcuts.json` (Linux). On Linux, keys need `xdotool` (X11) or `ydotool` (Wayland), capture needs `gnome-screenshot`, `spectacle`, `grim` + `wl-copy` or `maim` + `xclip` |
| Clipboard | ✅ send text to the PC (copy, open, type), see the PC clipboard, history |
| System | ✅ lock, sleep, restart, shut down (long press) |
| Settings | ✅ language (English by default, or French) and start page, saved on the tablet |
