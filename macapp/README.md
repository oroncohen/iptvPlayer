# IPTV Player — macOS installer

`IPTV Player.pkg` is a standard macOS installer package containing
`IPTV Player.app` — fully self-contained: the server is a single compiled
Go binary (universal, runs on both Apple Silicon and Intel Macs), bundled
together with the built web UI. **No Ruby, Go, Node, or anything else needs
to be installed on the machine that runs it** — that was the whole point of
moving off the original Ruby-based backend.

## Installing on another Mac

Copy `IPTV Player.pkg` to the other Mac (AirDrop, USB, cloud drive —
anything), then install it **via Terminal**, not by double-clicking:

```
sudo installer -pkg "/path/to/IPTV Player.pkg" -target /
```

Enter that Mac's admin password when prompted. This installs
`IPTV Player.app` into `/Applications`.

### Why Terminal, not double-click

This app isn't signed with a paid Apple Developer account ($99/year — out
of scope for a personal tool), so it isn't notarized. Double-clicking the
`.pkg` (or the `.app` afterward) routes through Gatekeeper, which blocks
unnotarized software — and on current macOS, the classic "right-click →
Open" bypass often doesn't reliably get past this for a `.pkg`/`.app` that
was never notarized at all (it's really meant for apps that just lack a
*specific* signature, not this case). Running `sudo installer` from
Terminal installs the package directly at the system level and does not go
through that same Gatekeeper app-launch check, so it just works.

If you'd still rather use Finder: double-click the `.pkg`, and if
Installer.app itself gets blocked, open **System Settings → Privacy &
Security**, scroll down, and look for an **"Open Anyway"** button next to
the block notice, then try again. The Terminal method above is more
reliable and doesn't depend on that button being available.

## After installing

Launch **IPTV Player** from `/Applications` (Spotlight, Launchpad, or
Finder) like any other app. It opens `http://localhost:4568` in your
default browser — paste a playlist URL there same as usual. Quitting the
app (Cmd+Q, or right-click the Dock icon → Quit) stops the bundled server
too.

## Rebuilding the installer after changing the app

```
cd ../frontend && npm run build     # picks up any frontend changes
cd ../macapp && ./build_installer.sh  # rebuilds the Go binary + repackages the .pkg
```

(Ruby is no longer used anywhere in this project — the whole backend was
rewritten in Go, specifically so it could ship as a single dependency-free
binary. See `../goserver/`.)

## Running in Docker instead

From the project root (the folder containing `Dockerfile`):

```
docker compose up -d --build
```

Then open http://localhost:62626 (set `IPTV_PORT=8080` to use another host
port). The image builds the web UI and the Go server itself, so nothing else
needs to be installed. The playlist history and cached channel lists are kept
in the `iptv-data` volume, so they survive `docker compose down` and image
rebuilds (only `docker compose down -v` deletes them).

## Playlist history and saved channel lists

Every playlist that loads successfully is remembered (up to 20, most recent
first) and shown under the URL box — click one to load it again, or × to
forget it. Its channel list is also saved, so reopening a known playlist is
instant and doesn't hit your provider; the **↻ Reload** button re-downloads
the latest list on demand (a failed reload keeps the saved copy).

This lives in `history.json` and `cache/` inside the data directory:
`$IPTV_DATA_DIR` if set (`/data` in Docker), otherwise
`~/Library/Application Support/IPTV Player`. Playlist URLs often contain your
provider username/password, so these files are readable only by your user.
