package backend

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ResolveModel turns a model name like "large-v3-turbo" into a path to the .bin file.
// Search order:
//  1. If model is already an absolute path, use it directly
//  2. ~/.local/share/whisper-cpp/ggml-<model>.bin
//  3. /opt/homebrew/share/whisper-cpp/ggml-<model>.bin
func ResolveModel(model string) (string, error) {
	if filepath.IsAbs(model) {
		if _, err := os.Stat(model); err == nil {
			return model, nil
		}
		return "", fmt.Errorf("model file not found: %s", model)
	}

	name := model
	if !strings.HasPrefix(name, "ggml-") {
		name = "ggml-" + name
	}
	if !strings.HasSuffix(name, ".bin") {
		name = name + ".bin"
	}

	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".local", "share", "whisper-cpp", name),
		filepath.Join("/opt/homebrew/share/whisper-cpp", name),
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("model %q not found — download with:\n  curl -L -o %s https://huggingface.co/ggerganov/whisper.cpp/resolve/main/%s",
		model, candidates[0], name)
}

// PreflightLocal checks that whisper-cli binary and model are available.
func PreflightLocal(model, binary string) (string, string, error) {
	if binary == "" {
		var err error
		binary, err = exec.LookPath("whisper-cli")
		if err != nil {
			return "", "", fmt.Errorf("whisper-cli not found — install with: brew install whisper-cpp")
		}
	}

	modelPath, err := ResolveModel(model)
	if err != nil {
		return "", "", err
	}

	return binary, modelPath, nil
}

func TranscribeLocal(audioFile, language, modelPath, binary string) (string, error) {
	args := []string{
		"-m", modelPath,
		"-l", language,
		"--no-timestamps",
		"--no-prints",
		"-f", audioFile,
	}

	cmd := exec.Command(binary, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("whisper-cli: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	return strings.TrimSpace(string(out)), nil
}
