package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/slauger/dictctl/internal/download"
	"gopkg.in/yaml.v3"
)

var languages = []struct {
	Code string
	Name string
}{
	{"en", "English"},
	{"de", "German"},
	{"fr", "French"},
	{"es", "Spanish"},
	{"it", "Italian"},
	{"pt", "Portuguese"},
	{"nl", "Dutch"},
	{"pl", "Polish"},
	{"ru", "Russian"},
	{"uk", "Ukrainian"},
	{"ja", "Japanese"},
	{"zh", "Chinese"},
	{"ko", "Korean"},
	{"ar", "Arabic"},
	{"tr", "Turkish"},
	{"sv", "Swedish"},
	{"da", "Danish"},
	{"no", "Norwegian"},
	{"fi", "Finnish"},
	{"cs", "Czech"},
	{"ro", "Romanian"},
	{"hu", "Hungarian"},
	{"el", "Greek"},
	{"he", "Hebrew"},
	{"hi", "Hindi"},
	{"th", "Thai"},
	{"vi", "Vietnamese"},
	{"id", "Indonesian"},
	{"ms", "Malay"},
	{"ca", "Catalan"},
	{"auto", "Auto-detect"},
}

var backends = []struct {
	Name string
	Desc string
}{
	{"local", "whisper-cpp (offline, local GPU)"},
	{"openai", "OpenAI Whisper API (cloud)"},
}

func fzfSelect(prompt string, items []string) (string, error) {
	cmd := exec.Command("fzf", "--prompt", prompt, "--height", "~20", "--reverse")
	cmd.Stdin = strings.NewReader(strings.Join(items, "\n"))
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func Run(currentLang, currentDevice, currentBackend, currentModel string) error {
	if _, err := exec.LookPath("fzf"); err != nil {
		return fmt.Errorf("fzf not found — install with: brew install fzf")
	}

	cfg := make(map[string]any)
	configPath := configFilePath()

	data, err := os.ReadFile(configPath)
	if err == nil {
		_ = yaml.Unmarshal(data, &cfg)
	}

	// 1. Language
	lang, err := selectLanguage(currentLang)
	if err != nil {
		return nil
	}
	if lang != "" {
		cfg["language"] = lang
		fmt.Fprintf(os.Stderr, "Language: %s\n", lang)
	}

	// 2. Backend
	be, err := selectBackend(currentBackend)
	if err != nil {
		return nil
	}
	if be != "" {
		cfg["default_backend"] = be
		fmt.Fprintf(os.Stderr, "Backend: %s\n", be)
	}

	// 3. Model (for local backend)
	model, err := selectModel(currentModel)
	if err != nil {
		return nil
	}
	if model != "" {
		backends, _ := cfg["backends"].(map[string]any)
		if backends == nil {
			backends = make(map[string]any)
		}
		local, _ := backends["local"].(map[string]any)
		if local == nil {
			local = make(map[string]any)
		}
		local["model"] = model
		backends["local"] = local
		cfg["backends"] = backends
		fmt.Fprintf(os.Stderr, "Model: %s\n", model)
	}

	// 4. Device
	dev, err := selectDevice(currentDevice)
	if err != nil {
		return nil
	}
	if dev != "" {
		cfg["device"] = dev
		fmt.Fprintf(os.Stderr, "Device: %s\n", dev)
	}

	// Write config
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(configPath, out, 0644); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "\nSaved to %s\n", configPath)
	return nil
}

func selectLanguage(current string) (string, error) {
	var items []string
	for _, l := range languages {
		marker := "  "
		if l.Code == current {
			marker = "* "
		}
		items = append(items, fmt.Sprintf("%s%-6s %s", marker, l.Code, l.Name))
	}

	selected, err := fzfSelect("Language: ", items)
	if err != nil {
		return "", err
	}

	fields := strings.Fields(selected)
	if len(fields) < 2 {
		return "", nil
	}
	code := fields[0]
	if code == "*" {
		code = fields[1]
	}
	return code, nil
}

func selectBackend(current string) (string, error) {
	var items []string
	for _, b := range backends {
		marker := "  "
		if b.Name == current {
			marker = "* "
		}
		items = append(items, fmt.Sprintf("%s%-8s %s", marker, b.Name, b.Desc))
	}

	selected, err := fzfSelect("Backend: ", items)
	if err != nil {
		return "", err
	}

	fields := strings.Fields(selected)
	if len(fields) < 2 {
		return "", nil
	}
	name := fields[0]
	if name == "*" {
		name = fields[1]
	}
	return name, nil
}

func selectModel(current string) (string, error) {
	var items []string
	for _, m := range download.AvailableModels {
		marker := "  "
		if m.Name == current {
			marker = "* "
		}
		installed := " "
		dest := filepath.Join(download.ModelDir(), download.ModelFileName(m.Name))
		if _, err := os.Stat(dest); err == nil {
			installed = "✓"
		}
		items = append(items, fmt.Sprintf("%s%s %-20s %s", marker, installed, m.Name, m.Size))
	}

	selected, err := fzfSelect("Model: ", items)
	if err != nil {
		return "", err
	}

	fields := strings.Fields(selected)
	if len(fields) < 2 {
		return "", nil
	}
	// Skip marker (*) and checkmark (✓)
	for _, f := range fields {
		if f != "*" && f != "✓" {
			return f, nil
		}
	}
	return "", nil
}

func selectDevice(current string) (string, error) {
	items := []string{"  (none)    Use system default"}

	cmd := exec.Command("system_profiler", "SPAudioDataType", "-json")
	out, err := cmd.Output()
	if err == nil {
		var data struct {
			Audio []struct {
				Items []struct {
					Name          string `json:"_name"`
					InputChannels int    `json:"coreaudio_device_input"`
				} `json:"_items"`
			} `json:"SPAudioDataType"`
		}
		if err := json.Unmarshal(out, &data); err == nil {
			for _, section := range data.Audio {
				for _, item := range section.Items {
					if item.InputChannels > 0 {
						marker := "  "
						if item.Name == current {
							marker = "* "
						}
						items = append(items, fmt.Sprintf("%s%s", marker, item.Name))
					}
				}
			}
		}
	}

	selected, err := fzfSelect("Audio device: ", items)
	if err != nil {
		return "", err
	}

	selected = strings.TrimLeft(selected, "* ")
	if strings.HasPrefix(selected, "(none)") {
		return "", nil
	}
	return selected, nil
}

func configFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "dictctl", "config.yaml")
}
