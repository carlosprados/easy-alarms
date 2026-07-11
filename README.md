# Easy Alarms

A simple alarm clock and timer for the Linux desktop, written in Go with
[Fyne](https://fyne.io). A lightweight clone of
[alarm-clock-applet](https://alarm-clock-applet.github.io) with one extra
focus: **the list tells you exactly when each alarm will ring.**

```
⏰ Despertador · 07:30
   🔔 Suena mañana a las 07:30 (en 10h 25m)
⏱ Pomodoro · 25m
   💤 Inactiva
```

## Features

- **Clock alarms** (HH:MM) with optional per-weekday repeat (L M X J V S D).
- **Countdown timers** (`10m`, `1h30m`, `45s`) with **pause/resume**.
- **Quick timers from the tray**: 5m / 10m / 25m / 1h, one click, no dialog.
- Every row shows **when it next rings** and the countdown, updated live.
- **Saving activates**: editing an alarm re-enables it; editing a timer
  (re)starts it. Enter in any field saves the dialog.
- **Per-alarm sound**: pick any audio file, or use the built-in beep. Preview
  it from the edit dialog with a single play/stop button. Alarms **fade in**
  over 10 s instead of blasting at full volume.
- **Snooze** (5 / 10 / 15 min) from the ringing dialog.
- An unattended alarm **auto-silences after 3 min** (the dialog stays up and
  a "missed alarm" notification is sent).
- **Duplicate** any alarm from its row to create variants quickly.
- **System-tray** icon; closing the window minimises to the tray.
- **Autostart on login** — toggle it from the tray menu.
- Survives system suspend: a missed alarm rings right after resume.
- **`alarmctl` CLI + MCP server**: control the running app from the terminal or
  from a local AI. See [Controlling from the CLI (alarmctl)](#controlling-from-the-cli-alarmctl).

## Requirements

- Linux desktop with a system tray / StatusNotifierItem host (on GNOME, the
  *AppIndicator* extension).
- Go 1.25+ and a C toolchain (Fyne needs cgo + OpenGL/X11 dev headers).
- [Task](https://taskfile.dev) for the build shortcuts (optional — plain
  `go build` works too).

## Build & run

```bash
task run            # build and launch with the window visible
task run-no-tray    # launch without the system tray (diagnostics)
task test           # run the unit tests
task install        # install into ~/.local (binary + icon + app menu entry)
task uninstall      # remove everything task install added
```

`task install` performs a per-user XDG install (no root needed):

| Artifact | Destination |
|----------|-------------|
| GUI binary | `~/.local/bin/easy-alarms` |
| CLI binary | `~/.local/bin/alarmctl` |
| Icon (SVG + PNG 48/64/128/256) | `~/.local/share/icons/hicolor/.../apps/` |
| App menu entry | `~/.local/share/applications/easy-alarms.desktop` |

After installing, **Easy Alarms** shows up in your application menu. Enable
*autostart on login* from the tray menu — the generated
`~/.config/autostart/easy-alarms.desktop` points at the installed binary, so
install first.

## Supported audio formats

`MP3`, `WAV` (8/16/24-bit PCM), `OGG`/`OGA` (Vorbis) and `FLAC`, decoded in
pure Go. 32-bit WAV is **not** supported — the preview will tell you if a file
can't be decoded.

### Converting audio to MP3

If a sound doesn't play (e.g. a 32-bit/96 kHz WAV from Freesound), convert it
with `ffmpeg`. This re-encodes to VBR ~190 kbps at 44.1 kHz and keeps the
original file:

```bash
ffmpeg -i input.wav -codec:a libmp3lame -q:a 2 -ar 44100 output.mp3
```

Batch-convert every WAV/FLAC in a folder:

```bash
for f in *.wav *.flac; do
  ffmpeg -i "$f" -codec:a libmp3lame -q:a 2 -ar 44100 "${f%.*}.mp3"
done
```

## Controlling from the CLI (alarmctl)

`alarmctl` is a second binary that drives a **running** easy-alarms app over a
local Unix socket (`$XDG_RUNTIME_DIR/easy-alarms/control.sock`, `0600`). It does
not launch the app; if the app isn't running, commands fail with a clear
message (exit code 2).

The command tree is fully self-documenting — `alarmctl <cmd> --help` shows flags
and examples for every command:

```bash
alarmctl status                                     # next alarm, anything ringing
alarmctl list [--kind clock|timer]                  # list everything

alarmctl alarm create --at 07:30 --days weekdays --label "Work"
alarmctl alarm create --at 09:00 --days lun,mié,vie # ES or EN day names
alarmctl alarm edit  <id> --at 08:00 --days weekend
alarmctl alarm enable|disable|delete <id>

alarmctl timer create --duration 25m --label Pomodoro
alarmctl timer create --duration 1h30m --paused
alarmctl timer start|pause|resume|stop <id>

alarmctl snooze [<id>] --for 10m                    # id optional if one is ringing
alarmctl dismiss [<id>]                             # no id = dismiss all ringing
```

`--days` accepts `daily` / `weekdays` / `weekend`, or a comma list of day names
in English (`mon,tue,…`) or Spanish (`lun,mar,mié,…`); omit it for a one-shot.
Add `--json` to any command for machine-readable output.

**Exit codes:** `0` success · `1` invalid input or conflict · `2` app not
running · `3` alarm not found.

### MCP server (for local AIs)

`alarmctl mcp` runs a [Model Context Protocol](https://modelcontextprotocol.io)
server on stdio, exposing tools (`create_alarm`, `create_timer`, `list_alarms`,
`timer_control`, `snooze`, `dismiss`, …), a resource (`alarms://all`) and prompts
(`wake-me-up`, `pomodoro`). Register it with Claude Code:

```bash
claude mcp add easy-alarms -- alarmctl mcp
```

Then just ask, e.g. *"ponme una alarma mañana a las 8 entre semana"* — the model
calls `create_alarm` and confirms the next ring time. The GUI app must be
running for the tools to take effect.

## Configuration

Alarms are stored as JSON at `~/.config/easy-alarms/alarms.json`. An empty or
corrupt file is tolerated — the app starts fresh and moves a bad file aside as
`alarms.json.corrupt` rather than refusing to launch.

The GUI accepts `--hidden` (start minimised to the tray), `--no-tray`
(diagnostic) and `--no-control` (disable the `alarmctl` control socket).

## Project layout

| Path | Responsibility |
|------|----------------|
| `cmd/easy-alarms` | GUI app entry point + control-server wiring |
| `cmd/alarmctl` | `alarmctl` CLI entry point |
| `internal/alarm` | Alarm model and the poll-based scheduler |
| `internal/store` | JSON persistence |
| `internal/audio` | Audio playback and the built-in beep |
| `internal/autostart` | XDG autostart `.desktop` entry |
| `internal/ui` | Fyne window, dialogs, tray and rendering |
| `internal/control` | IPC boundary: Unix-socket server, client, DTOs, parsing |
| `internal/cli` | Cobra command tree for `alarmctl` |
| `internal/mcpserver` | MCP server (tools, resource, prompts) |
| `internal/humanize` | Language-neutral duration formatting |
