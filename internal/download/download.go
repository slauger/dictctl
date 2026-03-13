package download

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const huggingfaceBase = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/"

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
