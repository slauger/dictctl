package download

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const huggingfaceBase = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/"

// Available models with approximate sizes
var AvailableModels = []struct {
	Name string
	Size string
}{
	{"tiny", "75 MB"},
	{"tiny.en", "75 MB"},
	{"base", "142 MB"},
	{"base.en", "142 MB"},
	{"small", "466 MB"},
	{"small.en", "466 MB"},
	{"medium", "1.5 GB"},
	{"medium.en", "1.5 GB"},
	{"large-v1", "2.9 GB"},
	{"large-v2", "2.9 GB"},
	{"large-v3", "2.9 GB"},
	{"large-v3-turbo", "1.5 GB"},
}

func ModelFileName(model string) string {
	name := model
	if !strings.HasPrefix(name, "ggml-") {
		name = "ggml-" + name
	}
	if !strings.HasSuffix(name, ".bin") {
		name = name + ".bin"
	}
	return name
}

func ModelDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "whisper-cpp")
}

// SelectModel uses fzf to interactively select a model.
// Returns the model name or empty string if cancelled.
func SelectModel() (string, error) {
	if _, err := exec.LookPath("fzf"); err != nil {
		return "", fmt.Errorf("fzf not found — specify model with -m flag")
	}

	var lines []string
	for _, m := range AvailableModels {
		installed := " "
		dest := filepath.Join(ModelDir(), ModelFileName(m.Name))
		if _, err := os.Stat(dest); err == nil {
			installed = "✓"
		}
		lines = append(lines, fmt.Sprintf("%s  %-20s %s", installed, m.Name, m.Size))
	}

	cmd := exec.Command("fzf", "--prompt", "Select model: ", "--height", "~20", "--reverse")
	cmd.Stdin = strings.NewReader(strings.Join(lines, "\n"))
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", nil // user cancelled
	}

	// Parse: "✓  large-v3-turbo       1.5 GB"
	selected := strings.TrimSpace(string(out))
	fields := strings.Fields(selected)
	if len(fields) < 2 {
		return "", nil
	}
	// Skip the checkmark field
	name := fields[1]
	return name, nil
}

func Model(model string) error {
	name := ModelFileName(model)
	dest := filepath.Join(ModelDir(), name)

	if err := os.MkdirAll(ModelDir(), 0755); err != nil {
		return err
	}

	if _, err := os.Stat(dest); err == nil {
		fmt.Fprintf(os.Stderr, "Model already exists: %s\n", dest)
		return nil
	}

	url := huggingfaceBase + name
	fmt.Fprintf(os.Stderr, "Downloading %s ...\n", name)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d (model %q may not exist)", resp.StatusCode, model)
	}

	tmp := dest + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	written, err := io.Copy(f, resp.Body)
	_ = f.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("download interrupted: %w", err)
	}

	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}

	fmt.Fprintf(os.Stderr, "Saved %s (%d MB)\n", dest, written/1024/1024)
	return nil
}
