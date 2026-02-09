package signal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultDeviceName = "Prism"
	DefaultConfigPath = ".local/share/signal-cli"
)

type Client struct {
	ConfigPath string
	enabled    bool
}

func NewClient() *Client {
	configPath := filepath.Join(os.Getenv("HOME"), DefaultConfigPath)

	enabled := false
	if _, err := exec.LookPath("signal-cli"); err == nil {
		enabled = true
	}

	return &Client{
		ConfigPath: configPath,
		enabled:    enabled,
	}
}

func (c *Client) IsEnabled() bool {
	return c.enabled
}

type Account struct {
	Number string
	UUID   string
}

func (c *Client) exec(args ...string) ([]byte, error) {
	if !c.enabled {
		return nil, fmt.Errorf("signal-cli not found in PATH")
	}

	baseArgs := []string{"--config", c.ConfigPath, "--output=json"}
	cmd := exec.Command("signal-cli", append(baseArgs, args...)...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("signal-cli: %s", strings.TrimSpace(stderr.String()))
		}
		return nil, err
	}

	return stdout.Bytes(), nil
}

func (c *Client) GetLinkedAccount() (*Account, error) {
	if c == nil || !c.enabled {
		return nil, nil
	}

	accountsFile := filepath.Join(c.ConfigPath, "data", "accounts.json")
	if _, err := os.Stat(accountsFile); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(accountsFile)
	if err != nil {
		return nil, err
	}

	var accountsData struct {
		Accounts []struct {
			Number string `json:"number"`
			UUID   string `json:"uuid"`
		} `json:"accounts"`
	}

	if err := json.Unmarshal(data, &accountsData); err != nil {
		return nil, err
	}

	if len(accountsData.Accounts) == 0 {
		return nil, nil
	}

	account := &Account{
		Number: accountsData.Accounts[0].Number,
		UUID:   accountsData.Accounts[0].UUID,
	}

	cmd := exec.Command("signal-cli", "-a", account.Number, "receive", "--timeout", "0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		errStr := strings.ToLower(string(output))
		if strings.Contains(errStr, "not registered") || strings.Contains(errStr, "authorization failed") {
			return nil, nil
		}
	}

	return account, nil
}

func (c *Client) CreateGroup(name string) (string, string, error) {
	if c == nil || !c.enabled {
		return "", "", fmt.Errorf("signal client not initialized")
	}

	account, err := c.GetLinkedAccount()
	if err != nil {
		return "", "", fmt.Errorf("failed to get linked account: %w", err)
	}
	if account == nil {
		return "", "", fmt.Errorf("no linked Signal account")
	}

	output, err := c.exec("-o", "json", "-a", account.Number, "updateGroup", "-n", name)
	if err != nil {
		return "", "", fmt.Errorf("failed to create group: %w", err)
	}

	var response struct {
		GroupID string `json:"groupId"`
	}

	lines := bytes.Split(output, []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		if err := json.Unmarshal(line, &response); err == nil && response.GroupID != "" {
			return response.GroupID, account.Number, nil
		}
	}

	return "", "", fmt.Errorf("failed to parse group creation response")
}

func (c *Client) SendGroupMessage(groupID, message string) error {
	if c == nil || !c.enabled {
		return fmt.Errorf("signal client not initialized")
	}

	account, err := c.GetLinkedAccount()
	if err != nil {
		return fmt.Errorf("failed to get linked account: %w", err)
	}
	if account == nil {
		return fmt.Errorf("no linked Signal account")
	}

	_, err = c.exec("-a", account.Number, "send", "-g", groupID, "--notify-self", "-m", message)
	return err
}

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
