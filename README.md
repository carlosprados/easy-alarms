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
- **Countdown timers** (`10m`, `1h30m`, `45s`).
- Every row shows **when it next rings** and the countdown, updated live.
- **Per-alarm sound**: pick any audio file, or use the built-in beep. Preview
  it from the edit dialog with a single play/stop button.
- **Snooze** (5 min) from the ringing dialog.
- **System-tray** icon; closing the window minimises to the tray.
- **Autostart on login** — toggle it from the tray menu.
- Survives system suspend: a missed alarm rings right after resume.

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
task install        # install to ~/.local/bin
```

> Install to `~/.local/bin` **before** enabling autostart, so the generated
> `~/.config/autostart/easy-alarms.desktop` points at a stable path rather
> than a throwaway build binary.

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

## Configuration

Alarms are stored as JSON at `~/.config/easy-alarms/alarms.json`. An empty or
corrupt file is tolerated — the app starts fresh and moves a bad file aside as
`alarms.json.corrupt` rather than refusing to launch.

## Project layout

| Path | Responsibility |
|------|----------------|
| `internal/alarm` | Alarm model and the poll-based scheduler |
| `internal/store` | JSON persistence |
| `internal/audio` | Audio playback and the built-in beep |
| `internal/autostart` | XDG autostart `.desktop` entry |
| `internal/ui` | Fyne window, dialogs, tray and rendering |
