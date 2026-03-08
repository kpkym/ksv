# ksv

A terminal UI (TUI) service manager built with Go. Start, stop, restart, and monitor multiple services from a single interactive dashboard.

Built with [Bubbletea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss).

## Screenshot

```
┌─ Services ─────────────┬─ Logs (my-app) ───────────────────────┐
│  ● my-app        [run] │  Server started on :8080              │
│  ○ worker       [stop] │  Connected to database                │
├────────────────────────┴───────────────────────────────────────┤
│ ↑↓/jk select  enter toggle  a start all  s stop all  q quit   │
└────────────────────────────────────────────────────────────────┘
```

## Features

- Start/stop/restart services individually or all at once
- Real-time log viewing per service
- Services keep running after quitting the TUI
- Process group management for clean shutdowns
- Automatic log file truncation when exceeding configured size
- PID file tracking for state persistence across TUI restarts

## Install

```bash
go install github.com/kpkym/ksv@latest
```

Or build from source:

```bash
make build    # outputs to dist/ksv
make install  # installs to $GOPATH/bin
```

## Configuration

Create a config file at `~/.config/ksv/services.conf`:

```ini
# Optional global settings
log_max_size = 10M

[my-app]
dir = /path/to/my-app
cmd = go run .

[worker]
dir = /path/to/worker
cmd = npm start
```

### Config file search order

1. `$KPK_CONFIG_DIR/ksv/services.conf`
2. `~/.config/ksv/services.conf`
3. `./services.conf`

### Options

| Setting | Scope | Description | Default |
|---------|-------|-------------|---------|
| `log_max_size` | Global | Max log file size (supports `K`, `M`, `G` suffixes, `0` for unlimited) | `10M` |
| `dir` | Service | Working directory | required |
| `cmd` | Service | Bash command to run | required |

## Keybindings

| Key | Action |
|-----|--------|
| `j` / `↓` | Next service |
| `k` / `↑` | Previous service |
| `Enter` | Toggle service (start/stop) |
| `r` | Restart service |
| `a` | Start all |
| `s` | Stop all |
| `G` | Jump to end of logs |
| `g` | Jump to start of logs |
| `q` | Quit (services keep running) |
| `Q` | Quit and stop all services |
