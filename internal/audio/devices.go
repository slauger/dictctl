package audio

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Device struct {
	Name      string
	Channels  int
	IsDefault bool
}

func ListInputDevices() ([]Device, error) {
	cmd := exec.Command("system_profiler", "SPAudioDataType", "-json")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("system_profiler failed: %w", err)
	}

	var data struct {
		Audio []struct {
			Items []struct {
				Name          string `json:"_name"`
				InputChannels int    `json:"coreaudio_device_input"`
				DefaultInput  string `json:"coreaudio_default_audio_input_device"`
			} `json:"_items"`
		} `json:"SPAudioDataType"`
	}
	if err := json.Unmarshal(out, &data); err != nil {
		return nil, fmt.Errorf("failed to parse audio data: %w", err)
	}

	var devices []Device
	for _, section := range data.Audio {
		for _, item := range section.Items {
			if item.InputChannels > 0 {
				devices = append(devices, Device{
					Name:      item.Name,
					Channels:  item.InputChannels,
					IsDefault: item.DefaultInput == "spaudio_yes",
				})
			}
		}
	}

	return devices, nil
}

func PrintDevices() error {
	devices, err := ListInputDevices()
	if err != nil {
		return err
	}

	if len(devices) == 0 {
		fmt.Println("No audio input devices found.")
		return nil
	}

	// If fzf is available and we have a TTY, offer interactive selection
	if _, fzfErr := exec.LookPath("fzf"); fzfErr == nil && isTerminal() {
		return selectDeviceInteractive(devices)
	}

	for _, d := range devices {
		marker := "  "
		if d.IsDefault {
			marker = "* "
		}
		fmt.Printf("%s%s (%d ch)\n", marker, d.Name, d.Channels)
	}
	fmt.Fprintln(os.Stderr, "\n* = default input device")
	fmt.Fprintf(os.Stderr, "Select with: dictctl -d \"<device name>\"\n")
	return nil
}

func selectDeviceInteractive(devices []Device) error {
	var lines []string
	for _, d := range devices {
		marker := " "
		if d.IsDefault {
			marker = "*"
		}
		lines = append(lines, fmt.Sprintf("%s %s (%d ch)", marker, d.Name, d.Channels))
	}

	cmd := exec.Command("fzf", "--prompt", "Select audio device: ", "--height", "~20", "--reverse")
	cmd.Stdin = strings.NewReader(strings.Join(lines, "\n"))
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil // user cancelled with Esc
	}

	selected := strings.TrimSpace(string(out))
	// Extract device name: strip leading "* " or "  " and trailing " (N ch)"
	selected = strings.TrimLeft(selected, "* ")
	if idx := strings.LastIndex(selected, " ("); idx != -1 {
		selected = selected[:idx]
	}

	if err := writeDeviceToConfig(selected); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Saved device %q to ~/.config/dictctl/config.yaml\n", selected)
	return nil
}

func writeDeviceToConfig(device string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".config", "dictctl")
	configPath := filepath.Join(configDir, "config.yaml")

	// Read existing config or start fresh
	var cfg map[string]any
	data, err := os.ReadFile(configPath)
	if err == nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return err
		}
	}
	if cfg == nil {
		cfg = make(map[string]any)
	}

	cfg["device"] = device

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(configPath, out, 0644)
}

func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// resolveDeviceIndex maps a device name to an avfoundation audio device index
// by parsing ffmpeg -list_devices output.
func resolveDeviceIndex(name string) (string, error) {
	cmd := exec.Command("ffmpeg", "-f", "avfoundation", "-list_devices", "true", "-i", "")
	out, err := cmd.CombinedOutput()
	// ffmpeg exits with error when listing devices, that's expected
	_ = err

	re := regexp.MustCompile(`\[AVFoundation.*\] \[(\d+)\] (.+)`)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	inAudio := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "AVFoundation audio devices") {
			inAudio = true
			continue
		}
		if !inAudio {
			continue
		}
		matches := re.FindStringSubmatch(line)
		if len(matches) == 3 {
			if matches[2] == name {
				return matches[1], nil
			}
		}
	}

	return "", fmt.Errorf("audio device %q not found — run 'dictctl devices' to list available devices", name)
}
