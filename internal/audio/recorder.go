package audio

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func Record(silenceDetection bool, device string) (string, error) {
	tmpFile, err := os.CreateTemp("", "dictctl-*.wav")
	if err != nil {
		return "", err
	}
	_ = tmpFile.Close()
	path := tmpFile.Name()

	var cmd *exec.Cmd
	useFFmpeg := device != ""
	if useFFmpeg {
		cmd, err = buildFFmpegCmd(device, path)
	} else {
		cmd, err = buildRecCmd(silenceDetection, path)
	}
	if err != nil {
		_ = os.Remove(path)
		return "", err
	}

	// Only show rec stderr (sox is quiet with -q); ffmpeg stderr is suppressed via -loglevel
	if !useFFmpeg {
		cmd.Stderr = os.Stderr
	}

	// For ffmpeg: pipe stdin so we can send 'q' for graceful stop
	var stdinPipe io.WriteCloser
	if useFFmpeg {
		stdinPipe, err = cmd.StdinPipe()
		if err != nil {
			_ = os.Remove(path)
			return "", err
		}
	}

	if useFFmpeg {
		fmt.Fprintf(os.Stderr, "Recording from %q... (press Ctrl+C to stop)\n", device)
	} else {
		fmt.Fprintln(os.Stderr, "Recording... (press Ctrl+C to stop)")
	}

	if err := cmd.Start(); err != nil {
		_ = os.Remove(path)
		return "", err
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)

	go func() {
		<-sigCh
		if useFFmpeg {
			// Send 'q' for graceful stop — ffmpeg finalizes the WAV header
			_, _ = stdinPipe.Write([]byte("q"))
		} else {
			_ = cmd.Process.Signal(syscall.SIGINT)
		}
	}()

	err = cmd.Wait()
	signal.Stop(sigCh)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				if status.Signaled() && status.Signal() == syscall.SIGINT {
					err = nil
				}
			}
			// rec returns exit code 2 on SIGINT
			if exitErr.ExitCode() == 2 {
				err = nil
			}
		}
	}

	if err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("recording failed: %w", err)
	}

	info, err := os.Stat(path)
	if err != nil || info.Size() < 100 {
		_ = os.Remove(path)
		return "", fmt.Errorf("recording too short or empty")
	}

	fmt.Fprintln(os.Stderr, "Recording stopped.")
	return path, nil
}

func buildRecCmd(silenceDetection bool, path string) (*exec.Cmd, error) {
	if _, err := exec.LookPath("rec"); err != nil {
		return nil, fmt.Errorf("rec (sox) not found — install with: brew install sox")
	}

	args := []string{"-q", "-r", "16000", "-c", "1", "-b", "16", path}
	if silenceDetection {
		args = append(args, "silence", "1", "0.1", "0.1%", "1", "2.0", "0.1%")
	}

	return exec.Command("rec", args...), nil
}

func buildFFmpegCmd(device, path string) (*exec.Cmd, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("ffmpeg not found — install with: brew install ffmpeg\n  (required for device selection, or remove 'device' from config to use default)")
	}

	// Resolve device name to avfoundation index
	index, err := resolveDeviceIndex(device)
	if err != nil {
		return nil, err
	}

	args := []string{
		"-loglevel", "error",
		"-f", "avfoundation",
		"-i", ":" + index,
		"-ar", "16000",
		"-ac", "1",
		"-y",
		path,
	}

	return exec.Command("ffmpeg", args...), nil
}
