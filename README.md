# dictctl

[![CI](https://github.com/slauger/dictctl/actions/workflows/ci.yaml/badge.svg)](https://github.com/slauger/dictctl/actions/workflows/ci.yaml)
[![Release](https://img.shields.io/github/v/release/slauger/dictctl)](https://github.com/slauger/dictctl/releases/latest)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/github/license/slauger/dictctl)](LICENSE)

CLI tool for dictation: microphone recording → Whisper transcription → text on stdout.

- 🎙️ **Record & Transcribe** — Record from your microphone, stop with Ctrl+C, get text on stdout
- 🏠 **Local Backend** — Offline transcription via [whisper.cpp](https://github.com/ggml-org/whisper.cpp) — no data leaves your machine
- ☁️ **OpenAI Backend** — Cloud transcription via the OpenAI Whisper API — no local GPU needed
- 📋 **Clipboard Support** — Copy transcription result directly to clipboard with `-c`
- 🔇 **Silence Detection** — Auto-stop recording after silence with `-s`
- 📁 **File Transcription** — Transcribe existing audio files without recording
- 🔧 **Interactive Setup** — Configure language, backend, model, API key, and audio device via `dictctl setup` (requires [fzf](https://github.com/junegunn/fzf))
- ⚡ **Single Binary** — No runtime dependencies, no Python, no Docker — just a Go binary and sox

## Requirements

- **sox** — audio recording (`rec` command)
- **whisper-cpp** — local transcription (optional, for `local` backend)
- **OpenAI API key** — cloud transcription (optional, for `openai` backend)
- **ffmpeg** — recording from a selected audio device (optional, macOS, see [Audio Devices](#audio-devices))
- **fzf** — interactive selection in `dictctl setup`, `dictctl download` and device picking (optional)

```bash
brew install sox whisper-cpp
brew install ffmpeg fzf   # optional
```

## Installation

Prebuilt binaries for macOS (arm64) and Linux (amd64, arm64) are attached to each [GitHub release](https://github.com/slauger/dictctl/releases/latest), together with `checksums.txt`:

```bash
curl -LO https://github.com/slauger/dictctl/releases/latest/download/dictctl-darwin-arm64
curl -LO https://github.com/slauger/dictctl/releases/latest/download/checksums.txt
shasum -a 256 --ignore-missing -c checksums.txt
install -m 755 dictctl-darwin-arm64 /usr/local/bin/dictctl
```

With Go:

```bash
go install github.com/slauger/dictctl/cmd/dictctl@latest
```

Or build from source:

```bash
git clone https://github.com/slauger/dictctl.git
cd dictctl
make install
```

## Quick Start

```bash
# Download the default whisper model (~1.5 GB)
dictctl download

# Start dictating (Ctrl+C to stop)
dictctl
```

## Usage

```bash
dictctl                         # record → default backend
dictctl -b openai               # record → OpenAI API
dictctl file audio.mp3          # transcribe existing file
dictctl devices                 # list audio input devices
dictctl download                # download whisper model (interactive)
dictctl download -m base        # download a specific model
dictctl setup                   # interactive configuration (language, backend, model, API key, device)
dictctl version                 # print version
dictctl --help                  # show help
```

Press **Ctrl+C** to stop recording. The audio is finalized cleanly and passed to the transcription backend.

### Flags

| Flag | Description |
|------|-------------|
| `-b <backend>` | Backend: `local`, `openai` (default: from config) |
| `-c` | Copy result to clipboard (macOS, via `pbcopy`) |
| `-d <device>` | Audio input device (see `dictctl devices`) |
| `-l <lang>` | Language code (default: `en`) |
| `-s` | Enable silence detection (auto-stop recording) |
| `-m <model>` | Override model name |
| `-h, --help` | Show help |
| `-v, --version` | Print version |

### Examples

```bash
# Record and transcribe in English (default)
dictctl

# Record in German via OpenAI
dictctl -b openai -l de

# Transcribe a file and copy to clipboard
dictctl file meeting.wav -c

# Use a specific local model
dictctl -m large-v3

# Record from a specific device
dictctl -d "Elgato Wave:3"
```

## Models

For the `local` backend, whisper.cpp GGML models are available. For the `openai` backend, the supported models are `whisper-1`, `gpt-4o-transcribe`, and `gpt-4o-mini-transcribe`.

Download a local model:

```bash
dictctl download                # downloads the configured model (default: large-v3-turbo)
dictctl download -m base        # download a smaller/faster model
```

Models are stored in `~/.local/share/whisper-cpp/`. The search order is:

1. `~/.local/share/whisper-cpp/ggml-<model>.bin`
2. `/opt/homebrew/share/whisper-cpp/ggml-<model>.bin`

Or specify an absolute path via config or `-m` flag.

## Audio Devices

List available input devices:

```bash
dictctl devices
```

```
* Elgato Wave:3 (1 ch)
  MacBook Pro-Mikrofon (1 ch)
  ...

* = default input device
```

Select a device per invocation with `-d` or set a default in the config file. When a device is configured, recording uses ffmpeg (avfoundation) instead of sox. Without a device, it uses the system default via sox.

## Configuration

Config file: `~/.config/dictctl/config.yaml`

```yaml
default_backend: local
language: en
# device: "Elgato Wave:3"

backends:
  local:
    model: large-v3-turbo
    # binary: /opt/homebrew/bin/whisper-cli
  openai:
    api_key: sk-...
    model: whisper-1
```

The OpenAI API key can also be set via the `OPENAI_API_KEY` environment variable or configured interactively with `dictctl setup`.

## License

MIT
