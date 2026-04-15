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

[pinger]
dir = /tmp
cmd = sudo ping -i 0.5 127.0.0.1
sudo = true
```

When `sudo = true`, ksv prompts for the password the first time the service starts, validates it via `sudo -S -v`, and refreshes the sudo timestamp every 4 minutes so the service stays runnable without re-prompting. The password is held only in memory and cleared when ksv exits.

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
| `sudo` | Service | Prompt for sudo password and keep timestamp fresh while running | `false` |

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
