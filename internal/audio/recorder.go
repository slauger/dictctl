package audio

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func Record(silenceDetection bool, device string) (string, error) {
	if _, err := exec.LookPath("rec"); err != nil {
		return "", fmt.Errorf("rec (sox) not found — install with: brew install sox")
	}

	tmpFile, err := os.CreateTemp("", "dictctl-*.wav")
	if err != nil {
		return "", err
	}
	_ = tmpFile.Close()
	path := tmpFile.Name()

	args := []string{"-q", "-r", "16000", "-c", "1", "-b", "16", path}
	if silenceDetection {
		args = append(args, "silence", "1", "0.1", "0.1%", "1", "2.0", "0.1%")
	}

	cmd := exec.Command("rec", args...)
	cmd.Stderr = os.Stderr
	if device != "" {
		cmd.Env = append(os.Environ(), "AUDIODEV="+device)
	}

	fmt.Fprintln(os.Stderr, "Recording... (press Ctrl+C to stop)")

	if err := cmd.Start(); err != nil {
		_ = os.Remove(path)
		return "", err
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)

	go func() {
		<-sigCh
		_ = cmd.Process.Signal(syscall.SIGINT)
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
			if exitErr.ExitCode() == 2 {
				err = nil
			}
		}
	}

	if err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("rec failed: %w", err)
	}

	info, err := os.Stat(path)
	if err != nil || info.Size() < 100 {
		_ = os.Remove(path)
		return "", fmt.Errorf("recording too short or empty")
	}

	fmt.Fprintln(os.Stderr, "Recording stopped.")
	return path, nil
}
