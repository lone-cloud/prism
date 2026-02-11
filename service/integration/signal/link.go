package signal

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func (c *Client) LinkDevice(deviceName string) (string, error) {
	if c == nil || !c.enabled {
		return "", fmt.Errorf("signal-cli not found in PATH")
	}

	if deviceName == "" {
		deviceName = DefaultDeviceName
	}

	if err := os.MkdirAll(c.ConfigPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	dataDir := filepath.Join(c.ConfigPath, "data")
	if entries, err := os.ReadDir(c.ConfigPath); err == nil {
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), "+") {
				accountDir := filepath.Join(c.ConfigPath, entry.Name())
				os.RemoveAll(accountDir)
			}
		}
	}
	os.RemoveAll(dataDir)

	cmd := exec.Command("signal-cli", "--config", c.ConfigPath, "link", "-n", deviceName)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start link command: %w", err)
	}

	var output bytes.Buffer
	buf := make([]byte, 8192)

	done := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				output.Write(buf[:n])
				text := output.String()

				if idx := strings.Index(text, "sgnl://linkdevice"); idx != -1 {
					end := strings.IndexAny(text[idx:], " \n\r\t")
					var url string
					if end == -1 {
						url = text[idx:]
					} else {
						url = text[idx : idx+end]
					}
					done <- strings.TrimSpace(url)
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					errChan <- err
				}
				return
			}
		}
	}()

	go func() {
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				output.Write(buf[:n])
				text := output.String()

				if idx := strings.Index(text, "sgnl://linkdevice"); idx != -1 {
					end := strings.IndexAny(text[idx:], " \n\r\t")
					var url string
					if end == -1 {
						url = text[idx:]
					} else {
						url = text[idx : idx+end]
					}
					done <- strings.TrimSpace(url)
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	select {
	case url := <-done:
		go func() {
			io.Copy(io.Discard, stdout)
			io.Copy(io.Discard, stderr)
			cmd.Wait()
		}()
		return url, nil
	case err := <-errChan:
		cmd.Process.Kill()
		return "", err
	case <-time.After(5 * time.Minute):
		cmd.Process.Kill()
		return "", fmt.Errorf("timeout waiting for QR code URL")
	}
}
