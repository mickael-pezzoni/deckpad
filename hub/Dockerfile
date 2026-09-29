# Image du hub deckpad (appli tablette incluse). Construire depuis la racine du dépôt :
#   docker build -f hub/Dockerfile -t deckpad-hub .
# ou, depuis hub/ : docker compose up -d
FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
# Le build écrit dans ../hub/webdist/dist
RUN npm run build

FROM golang:1.24-alpine AS build
WORKDIR /src/hub
COPY hub/go.mod hub/go.sum ./
RUN go mod download
COPY hub/ ./
COPY --from=web /src/hub/webdist/dist ./webdist/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /deckpad-hub . && mkdir /data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /deckpad-hub /deckpad-hub
# Appairages et certificat : à garder d'une mise à jour à l'autre (volume).
COPY --from=build --chown=nonroot:nonroot /data /data
VOLUME /data
EXPOSE 8430 8431
ENTRYPOINT ["/deckpad-hub", "-data", "/data"]
