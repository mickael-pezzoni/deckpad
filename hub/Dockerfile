# Image du serveur central deckpad. Construire depuis le dossier hub/ :
#   docker build -t deckpad-hub .
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /deckpad-hub .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /deckpad-hub /deckpad-hub
EXPOSE 8430
ENTRYPOINT ["/deckpad-hub"]
