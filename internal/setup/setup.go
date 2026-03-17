package setup

import (
	"bufio"
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

var openaiModels = []struct {
	Name string
	Desc string
}{
	{"whisper-1", "Classic Whisper model"},
	{"gpt-4o-transcribe", "GPT-4o based transcription"},
	{"gpt-4o-mini-transcribe", "GPT-4o Mini based transcription"},
}

func fzfSelect(prompt string, items []string) (string, error) {
	args := []string{"--prompt", prompt, "--height", "~20", "--reverse"}
	// Auto-focus on the currently selected item (marked with *)
	for i, item := range items {
		if strings.HasPrefix(item, "* ") {
			args = append(args, "--bind", fmt.Sprintf("load:pos(%d)", i+1))
			break
		}
	}
	cmd := exec.Command("fzf", args...)
	cmd.Stdin = strings.NewReader(strings.Join(items, "\n"))
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func Run(currentLang, currentDevice, currentBackend, currentModel, currentOpenAIModel, currentAPIKey string) error {
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

	// Determine effective backend
	effectiveBackend := currentBackend
	if be != "" {
		effectiveBackend = be
	}

	// 3. Model (backend-dependent)
	backendsMap, _ := cfg["backends"].(map[string]any)
	if backendsMap == nil {
		backendsMap = make(map[string]any)
	}

	switch effectiveBackend {
	case "openai":
		model, err := selectOpenAIModel(currentOpenAIModel)
		if err != nil {
			return nil
		}
		if model != "" {
			openai, _ := backendsMap["openai"].(map[string]any)
			if openai == nil {
				openai = make(map[string]any)
			}
			openai["model"] = model
			backendsMap["openai"] = openai
			cfg["backends"] = backendsMap
			fmt.Fprintf(os.Stderr, "Model: %s\n", model)
		}

		// 4. API Key (only for openai)
		apiKey := promptAPIKey(currentAPIKey)
		if apiKey != "" {
			openai, _ := backendsMap["openai"].(map[string]any)
			if openai == nil {
				openai = make(map[string]any)
			}
			openai["api_key"] = apiKey
			backendsMap["openai"] = openai
			cfg["backends"] = backendsMap
			fmt.Fprintf(os.Stderr, "API Key: %s***\n", apiKey[:4])
		}
	default:
		model, err := selectModel(currentModel)
		if err != nil {
			return nil
		}
		if model != "" {
			local, _ := backendsMap["local"].(map[string]any)
			if local == nil {
				local = make(map[string]any)
			}
			local["model"] = model
			backendsMap["local"] = local
			cfg["backends"] = backendsMap
			fmt.Fprintf(os.Stderr, "Model: %s\n", model)
		}
	}

	// 5. Device
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

func selectOpenAIModel(current string) (string, error) {
	var items []string
	for _, m := range openaiModels {
		marker := "  "
		if m.Name == current {
			marker = "* "
		}
		items = append(items, fmt.Sprintf("%s%-25s %s", marker, m.Name, m.Desc))
	}

	selected, err := fzfSelect("Model: ", items)
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

func promptAPIKey(currentKey string) string {
	envKey := os.Getenv("OPENAI_API_KEY")

	if currentKey != "" || envKey != "" {
		var items []string
		if currentKey != "" {
			items = append(items, fmt.Sprintf("Keep existing API key (%s***)", currentKey[:4]))
		}
		if envKey != "" && envKey != currentKey {
			items = append(items, fmt.Sprintf("Keep from env OPENAI_API_KEY (%s***)", envKey[:4]))
		}
		items = append(items, "Enter new API key")

		selected, err := fzfSelect("API Key: ", items)
		if err != nil {
			return ""
		}

		if strings.HasPrefix(selected, "Keep existing") {
			return ""
		}
		if strings.HasPrefix(selected, "Keep from env") {
			return envKey
		}
	}

	fmt.Fprint(os.Stderr, "Enter OpenAI API key: ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		key := strings.TrimSpace(scanner.Text())
		if key != "" {
			return key
		}
	}
	return ""
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
