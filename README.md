# deckpad

Piloter son PC depuis une tablette pendant qu'on joue : des pages par thème, en swipe, avec de grosses tuiles.

- `agent/` : programme Go qui tourne sur le PC (API + appli tablette embarquée), un seul `.exe`.
- `web/` : appli tablette (React + Vite + Swiper), installable en PWA.

## Utiliser

1. Lancer `deckpad.exe` sur le PC. Il affiche l'adresse à ouvrir (ex : `http://192.168.1.20:8420`).
2. Ouvrir cette adresse sur la tablette, puis « Ajouter à l'écran d'accueil ».

Au premier lancement, Windows demande d'autoriser l'accès réseau : accepter pour les réseaux privés.

> ⚠️ Pas encore d'authentification : à n'utiliser que sur un réseau local de confiance.

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
| Processus | à venir |
| Fichiers | à venir |
| Réseau | à venir |
| Système | à venir |
| Paramètres | à venir |
