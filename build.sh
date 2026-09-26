#!/usr/bin/env sh
# Construit deckpad.exe (appli tablette incluse) depuis macOS/Linux. Usage : ./build.sh
set -e
(cd web && npm ci && npm run build)
(cd agent && GOOS=windows GOARCH=amd64 go build -o ../bin/deckpad.exe .)
echo "OK : bin/deckpad.exe"
