# IPTV Player

A local IPTV player with a Go backend and a React frontend. Supports playlist loading, HLS playback, favorites, playlist history, and cached channel lists.

## Run with Docker

```sh
docker compose up -d --build
```

Open http://localhost:62626. Saved data is kept in the `iptv-data` Docker volume.

## Source layout

- `goserver/`: Go server and tests.
- `frontend/`: React/Vite interface and tests.
- `macapp/`: macOS installer build script and installation documentation.

## Build manually

Install Node.js and Go matching the versions used in `Dockerfile` and `goserver/go.mod`, then run:

```sh
cd frontend
npm ci
npm run build
cd ../goserver
go build -o ../iptvplayer .
cd ..
./iptvplayer 4568
```

Open http://localhost:4568.

The existing `start.sh` and macOS installer script use a local Go toolchain under `.gotoolchain/`; that toolchain is not included in this repository. See [macOS instructions](macapp/README.md).

## Tests

```sh
(cd goserver && go test ./...)
(cd frontend && npm test)
```

Generated binaries, installers, dependencies, and personal playlist data are excluded from version control.
