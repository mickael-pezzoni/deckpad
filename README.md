# deckpad

Control your PC from a tablet while you play: themed pages you swipe through, with big tiles.

- `hub/`: central server, hosted on an always-on machine at home. It finds the PCs running deckpad on the local network, serves the tablet app and relays its requests to the chosen PC. Docker image or single binary.
- `agent/`: Go program running on each PC (Windows or Linux), a single `.exe`. It only exposes an API to the hub.
- `web/`: tablet app (React + Vite + Swiper), embedded in the hub, installable as a PWA. Available in English and French.

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
| ![PC list](docs/screenshots/pcs.png) Choose a PC (hub) | |

On phones too, with a side menu:

<p>
  <img src="docs/screenshots/stats-phone.png" alt="Stats on a phone" width="240">
  <img src="docs/screenshots/network-phone.png" alt="Network on a phone" width="240">
  <img src="docs/screenshots/menu-phone.png" alt="Side menu on a phone" width="240">
</p>

## Usage

1. Start the hub on the home server (see below). It shows its address, e.g. `http://192.168.1.10:8430`.
2. Run `deckpad.exe` on each PC. Nothing to configure: the hub finds it on its own.
3. Open the hub's address on the tablet, pick a PC and type the 6-digit code shown on that PC's screen.

### Start the hub

With Docker, from the `hub/` folder:

```
docker compose up -d
```

The container uses the host network (`network_mode: host`), otherwise it can't hear the PCs' announcements. Pairings and the certificate are kept in a Docker volume. Without Docker: `cd hub && go run .` (Linux or Windows), data in `~/.config/deckpad-hub` (`%APPDATA%\deckpad-hub` on Windows), or `-data <folder>`.

The hub listens on port 8430 (HTTP) and 8431 (HTTPS). Open them in the server's firewall if needed (on NixOS: `networking.firewall.allowedTCPPorts = [ 8430 8431 ];`).

### How PCs are found

Each agent announces itself over mDNS (`_deckpad._tcp`), like printers do. The hub lists them, and PCs already paired stay listed (greyed out) while they are off.

- A PC disappears from the list about a minute after deckpad stops.
- Each agent keeps a stable id in `agent-id`, next to `devices.json`, so a PC whose address changes isn't listed twice.
- `-announce=false` on the agent turns the announcement off. If the Windows firewall asks, allow deckpad on private networks.

### Pair

Picking a PC that isn't paired yet opens a keypad on the tablet and a small window on the PC with a 6-digit code (also printed in the console). That one code does two things: it lets the tablet use the hub (the first time), and gives the hub a key to that PC. Other tablets pair the same way, with any PC's code.

- The code expires after 5 minutes and locks after 5 wrong attempts.
- The hub talks to each PC over HTTPS. The agent creates its own certificate on first launch (`agent.crt`, valid 20 years). The hub remembers its fingerprint at pairing and then refuses any other certificate, like SSH does: nothing to install on the PC or the tablet for this part.
- The PC only keeps the fingerprint of the hub's key, in `%APPDATA%\deckpad\devices.json` (Windows) or `~/.config/deckpad/devices.json` (Linux). Deleting this file unpairs the hub; it then asks for a new code.
- The window opens with Edge or Chrome (Chromium on Linux), otherwise in the default browser.
- The chosen PC is remembered on the tablet: the app reopens on it.

### Navigate

- Swipe between pages, or use the bottom menu. Its first button (two arrows) brings back the PC list.
- On a phone, the bottom bar only shows the current page: tap it to open a side menu with every page, "Switch PC" and the theme button.

### Install the app on the tablet (PWA)

Browsers only install a real app (full screen, icon, cache) over HTTPS. So the hub also serves the app over HTTPS on port 8431, with its own certificate.

Right after pairing, the tablet offers to switch to a secure connection, in 3 steps (only once):

1. "Download the certificate".
2. Android: in Settings, search for "CA certificate" and pick `deckpad-ca.crt`. iPad: Settings › Profile Downloaded › Install, then Settings › General › About › Certificate Trust Settings, turn on deckpad.
3. "Continue securely": the tablet switches to HTTPS without pairing again. All that's left is installing the app (⋮ menu → "Install app" in Chrome).

Good to know:

- On first launch, the hub creates its own small certificate authority in its data folder: `ca.crt` and `ca.key`. The key never leaves the server. It can only sign local network addresses, so it would be useless to impersonate another website.
- If the server's address changes, the certificate is regenerated automatically, with nothing to reinstall on the tablet.
- Android then shows "network may be monitored": this is normal after installing a certificate.
- `-https-addr ""` disables HTTPS. In development mode, HTTPS serves the built app: run `npm run build` before testing this step.

Without a certificate, on Android, you can also enable `chrome://flags/#unsafely-treat-insecure-origin-as-secure` with the hub's HTTP address, then install the app.

## Build

Requirements: [Go](https://go.dev/dl/) and [Node.js](https://nodejs.org/).

```
.\build.ps1      # Windows
./build.sh       # macOS / Linux
```

Output: `bin/deckpad.exe` (agent for Windows) and the hub (`bin/deckpad-hub`, app included); `build.sh` also builds the Linux agent `bin/deckpad`. Docker image: `docker build -f hub/Dockerfile -t deckpad-hub .` from the repository root.

## Develop

Three terminals:

```
cd agent && go run .        # API on :8421 (HTTPS), for the hub
cd hub && go run .          # finds the agent, relays the API, on :8430
cd web && npm install && npm run dev
```

Open the address shown by Vite (on the PC or the tablet). The app reloads on every save and `/api` calls are proxied to the hub, which relays them to the chosen PC (`/api/pc/<id>/…` → `/api/…` of the agent).

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
| Settings | ✅ language (the device's language on first launch, English if it is neither English nor French) and start page, saved on the tablet |
