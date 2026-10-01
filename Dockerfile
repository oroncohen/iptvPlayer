# IPTV Player — multi-stage image: build the web UI, build the Go server,
# ship both in a small Alpine runtime. Build context is the repo root.
#
#   docker compose up -d --build        (see docker-compose.yml)
#   -> http://localhost:62626

# ---- 1. web UI (Vite writes to ../public) ----
FROM node:24-alpine AS ui
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# ---- 2. Go server (static binary, no cgo) ----
FROM golang:1.27-alpine AS server
WORKDIR /src/goserver
COPY goserver/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/iptvplayer .

# ---- 3. runtime ----
FROM alpine:3.22
# ca-certificates: the server fetches https:// playlists and streams itself.
RUN apk add --no-cache ca-certificates \
 && adduser -D -u 10001 iptv \
 && mkdir /data && chown iptv /data
WORKDIR /app
# The server finds its web UI in a public/ folder next to the binary.
COPY --from=server /out/iptvplayer ./iptvplayer
COPY --from=ui /src/public ./public

USER iptv
# Saved playlist history + cached channel lists live here; mount a volume so
# they survive rebuilds/recreates of the container.
ENV IPTV_DATA_DIR=/data
VOLUME /data
EXPOSE 4568
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
  CMD wget -q -O /dev/null http://127.0.0.1:4568/ || exit 1
CMD ["./iptvplayer", "4568"]
