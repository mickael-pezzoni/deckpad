# deckpad

Piloter son PC depuis une tablette pendant qu'on joue : des pages par thème, en swipe, avec de grosses tuiles.

- `agent/` : programme Go qui tourne sur le PC (API + appli tablette embarquée), un seul `.exe`.
- `web/` : appli tablette (React + Vite + Swiper), installable en PWA.

## Utiliser

1. Lancer `deckpad.exe` sur le PC. Il affiche l'adresse à ouvrir (ex : `http://192.168.1.20:8420`).
2. Ouvrir cette adresse sur la tablette, puis « Ajouter à l'écran d'accueil ».

Au premier lancement, Windows demande d'autoriser l'accès réseau : accepter pour les réseaux privés.

### Appairer une tablette

La première fois, la tablette affiche un pavé numérique et le PC ouvre une petite fenêtre avec un code à 6 chiffres (aussi écrit dans la console). Taper ce code sur la tablette : elle reçoit une clé et n'aura plus à le refaire.

- Le code expire après 5 minutes et se bloque après 5 erreurs.
- Chaque appareil a sa propre clé. Le PC n'en garde que l'empreinte, dans `%APPDATA%\deckpad\devices.json` (Windows) ou `~/.config/deckpad/devices.json` (Linux). Supprimer ce fichier désappaire tout.
- La fenêtre s'ouvre avec Edge ou Chrome (Chromium sous Linux), sinon dans le navigateur par défaut.

## Construire l'exe

Prérequis : [Go](https://go.dev/dl/) et [Node.js](https://nodejs.org/).

```
.\build.ps1      # Windows
./build.sh       # macOS / Linux
```

Résultat : `bin/deckpad.exe`.

## Développer

Deux terminaux :

```
cd agent && go run .        # API sur :8420
cd web && npm install && npm run dev
```

Ouvrir l'adresse affichée par Vite (sur le PC ou la tablette). L'appli se recharge à chaque sauvegarde et les appels `/api` sont redirigés vers l'agent.

## Pages

| Page | État |
|---|---|
| Infos PC | ✅ |
| Stats | ✅ CPU, RAM, GPU toutes marques (FPS et température GPU AMD sous Windows à venir) |
| Processus | ✅ applications de l'utilisateur, tri CPU/RAM, fermeture avec confirmation |
| Fichiers | à venir |
| Réseau | ✅ débit en direct, latence, IP locale et publique, type de connexion |
| Audio | ✅ volume général et par appli, micro, choix de la sortie (casque, enceintes…). Sous Linux : `pactl` requis (fourni avec PulseAudio / PipeWire) |
| Système | ✅ verrouiller, veille, redémarrer, éteindre (appui long) |
| Paramètres | à venir |
