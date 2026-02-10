# Wu Bot Go - Usage

## Running

### Headless mode (server/background)
```bash
docker compose up -d          # start in background
docker compose logs -f        # follow logs
docker compose down           # stop
```

### TUI mode (interactive)
```bash
docker compose run --rm wubot --config /app/config.yaml
```

### Rebuild after code changes
```bash
docker compose build && docker compose up -d
```

## Config
Edit `config.yaml` (copied from `config.example.yaml`).

- `auto_start: true` - bot starts automatically in headless mode
- Press `s` in TUI to start/stop a bot manually

## TUI Keys
| Key     | Bot List          | Bot Detail        |
|---------|-------------------|-------------------|
| `s`     | Start bot         | Start/stop toggle |
| `x`     | Stop bot          |                   |
| `enter` | View bot detail   |                   |
| `j/k`   | Navigate list     | Scroll logs       |
| `?`     | Help              |                   |
| `esc`   | -                 | Back to list      |
| `q`     | Quit              | Back to list      |
