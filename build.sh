#!/usr/bin/env sh
# Construit le hub (appli tablette incluse) et l'agent pour Windows et Linux,
# depuis macOS/Linux. Usage : ./build.sh
set -e
(cd web && npm ci && npm run build)
(cd hub && go build -o ../bin/deckpad-hub .)
(cd agent && GOOS=windows GOARCH=amd64 go build -o ../bin/deckpad.exe .)
(cd agent && GOOS=linux GOARCH=amd64 go build -o ../bin/deckpad .)
echo "OK : bin/deckpad-hub (hub), bin/deckpad.exe et bin/deckpad (agent)"
