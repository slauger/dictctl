package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/slauger/dictctl/internal/audio"
	"github.com/slauger/dictctl/internal/backend"
	"github.com/slauger/dictctl/internal/clipboard"
	"github.com/slauger/dictctl/internal/config"
	"github.com/slauger/dictctl/internal/download"
	"github.com/slauger/dictctl/internal/setup"
)

var version = "dev"

type options struct {
	backend   string
	file      string
	clipboard bool
	language  string
	silence   bool
	model     string
	device    string
	devices   bool
	download  bool
	setup     bool
	version   bool
	help      bool
}

func parseArgs(args []string) options {
	opts := options{}
	i := 0
	for i < len(args) {
		switch args[i] {
		case "-c":
			opts.clipboard = true
		case "-l":
			i++
			if i < len(args) {
				opts.language = args[i]
			}
		case "-s":
			opts.silence = true
		case "-m":
			i++
			if i < len(args) {
				opts.model = args[i]
			}
		case "-b":
			i++
			if i < len(args) {
				opts.backend = args[i]
			}
		case "-d":
			i++
			if i < len(args) {
				opts.device = args[i]
			}
		case "file":
			i++
			if i < len(args) {
				opts.file = args[i]
			}
		case "devices":
			opts.devices = true
		case "download":
			opts.download = true
		case "setup":
			opts.setup = true
		case "version", "--version", "-v":
			opts.version = true
		case "help", "--help", "-h":
			opts.help = true
		default:
			fmt.Fprintf(os.Stderr, "dictctl: unknown argument: %s\n\n", args[i])
			fmt.Print(usage())
			os.Exit(1)
		}
		i++
	}
	return opts
}

func usage() string {
	return `Usage: dictctl [flags]
       dictctl <command> [flags]

Commands:
  file <path>   Transcribe an existing audio file
  devices       List audio input devices
  download      Download whisper model
  setup         Interactive configuration (requires fzf)
  version       Print version

Flags:
  -b <backend>  Backend: local, openai (default: from config)
  -c            Copy result to clipboard
  -l <lang>     Language code (default: en)
  -s            Enable silence detection
  -m <model>    Override model name
  -d <device>   Audio input device (see 'dictctl devices')
  -h, --help    Show this help
`
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "dictctl: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	opts := parseArgs(os.Args[1:])

	if opts.help {
		fmt.Print(usage())
		return
	}

	if opts.version {
		fmt.Println("dictctl " + version)
		return
	}

	if opts.devices {
		if err := audio.PrintDevices(); err != nil {
			fatal("%v", err)
		}
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fatal("config: %v", err)
	}

	if opts.setup {
		if err := setup.Run(cfg.Language, cfg.Device, cfg.DefaultBackend, cfg.Backends.Local.Model, cfg.Backends.OpenAI.Model, cfg.Backends.OpenAI.APIKey); err != nil {
			fatal("%v", err)
		}
		return
	}

	if opts.language != "" {
		cfg.Language = opts.language
	}

	backendName := cfg.DefaultBackend
	if opts.backend != "" {
		backendName = opts.backend
	}

	// Handle download subcommand
	if opts.download {
		model := opts.model
		if model == "" {
			// Try fzf interactive selection
			selected, err := download.SelectModel()
			if err != nil {
				// No fzf, fall back to configured default
				model = cfg.Backends.Local.Model
			} else if selected == "" {
				return // user cancelled
			} else {
				model = selected
			}
		}
		if err := download.Model(model); err != nil {
			fatal("%v", err)
		}
		return
	}

	// Preflight checks before recording
	var localBinary, localModelPath string
	if backendName == "local" {
		model := cfg.Backends.Local.Model
		if opts.model != "" {
			model = opts.model
		}
		localBinary, localModelPath, err = backend.PreflightLocal(model, cfg.Backends.Local.Binary)
		if err != nil {
			fatal("%v\n  Run: dictctl download", err)
		}
	}
	if backendName == "openai" {
		apiKey := cfg.Backends.OpenAI.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		if apiKey == "" {
			fatal("openai: no API key configured (set backends.openai.api_key or OPENAI_API_KEY)")
		}
	}

	// Determine device
	device := cfg.Device
	if opts.device != "" {
		device = opts.device
	}

	// Determine audio file
	var audioFile string
	if opts.file != "" {
		audioFile = opts.file
		if _, err := os.Stat(audioFile); err != nil {
			fatal("file not found: %s", audioFile)
		}
	} else {
		audioFile, err = audio.Record(opts.silence, device)
		if err != nil {
			fatal("recording: %v", err)
		}
		defer func() { _ = os.Remove(audioFile) }()
	}

	// Transcribe
	var text string
	switch backendName {
	case "local":
		text, err = backend.TranscribeLocal(audioFile, cfg.Language, localModelPath, localBinary)
	case "openai":
		model := cfg.Backends.OpenAI.Model
		if opts.model != "" {
			model = opts.model
		}
		apiKey := cfg.Backends.OpenAI.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		text, err = backend.TranscribeOpenAI(audioFile, cfg.Language, model, apiKey)
	default:
		fatal("unknown backend: %s", backendName)
	}
	if err != nil {
		fatal("transcription: %v", err)
	}

	text = strings.TrimSpace(text)
	if text == "" {
		fatal("no transcription result")
	}

	fmt.Println(text)

	if opts.clipboard {
		if err := clipboard.Copy(text); err != nil {
			fatal("clipboard: %v", err)
		}
		fmt.Fprintln(os.Stderr, "(copied to clipboard)")
	}
}
