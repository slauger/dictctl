package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

type audioDevice struct {
	Name      string
	Channels  int
	IsDefault bool
}

func listInputDevices() ([]audioDevice, error) {
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

	var devices []audioDevice
	for _, section := range data.Audio {
		for _, item := range section.Items {
			if item.InputChannels > 0 {
				devices = append(devices, audioDevice{
					Name:      item.Name,
					Channels:  item.InputChannels,
					IsDefault: item.DefaultInput == "spaudio_yes",
				})
			}
		}
	}

	return devices, nil
}

func printDevices() error {
	devices, err := listInputDevices()
	if err != nil {
		return err
	}

	if len(devices) == 0 {
		fmt.Println("No audio input devices found.")
		return nil
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
