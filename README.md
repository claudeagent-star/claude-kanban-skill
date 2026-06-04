# Claude Kanban Skill

A small self-hosted kanban board for Claude Code, bundled with a Go HTTP server
and a drag-and-drop frontend. State persists to a JSON file. No database, no
dependencies, no external services.

The skill lets Claude add, move, list, and delete cards on the board for you,
while you drag them around in a browser.

## Five columns, fixed

To Do · Blocked · In Progress · In Review · Done

## Installation

### From the Claude Code command line (recommended)

Register this repo as a plugin marketplace, then install:

```
/plugin marketplace add mdodkins/claude-kanban-skill
/plugin install kanban@mdodkins-kanban
```

### Manual installation

Copy the skill file into your personal skills folder:

```bash
mkdir -p ~/.claude/skills/kanban
cp skills/kanban/SKILL.md ~/.claude/skills/kanban/SKILL.md
```

You'll also need the Go server (next section).

## Running the server

The server is a single Go binary in `server/`. It embeds its own static frontend.

```bash
cd server
go build -o kanban .
./kanban --listen 127.0.0.1:8765 --state ~/.kanban/state.json
```

Then open http://127.0.0.1:8765/ in your browser. The state file is created on
first write.

Flags:

| Flag        | Default                       | What                              |
|-------------|-------------------------------|-----------------------------------|
| `--listen`  | `127.0.0.1:8765`              | host:port to bind                 |
| `--state`   | `~/.kanban/state.json`        | JSON state file path              |

## HTTP API

For when you (or Claude) want to script the board:

```
GET    /api/cards               list all cards
POST   /api/cards               create card (JSON: title, description, column)
PATCH  /api/cards/{id}          sparse update (any subset of fields)
DELETE /api/cards/{id}          remove card
```

Column IDs (use these literally): `to-do`, `blocked`, `in-progress`,
`in-review`, `done`.

## Running as a systemd service

Optional. Useful if you want the board running 24/7 on a server.

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin kanban
sudo mkdir -p /var/lib/kanban
sudo chown kanban:kanban /var/lib/kanban
sudo cp server/kanban /usr/local/bin/
sudo cp server/kanban.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now kanban
```

## State file

Plain JSON, one array of cards. Safe to back up, edit by hand (when the server
is stopped), or commit to git if the cards aren't sensitive.

## License

Whatever the parent project license is. (TBD; assume MIT for now unless the
owner says otherwise.)
