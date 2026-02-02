package signal

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type Daemon struct {
	binaryPath string
	dataPath   string
	socketPath string
	cmd        *exec.Cmd
}

func NewDaemon(binaryPath, dataPath, socketPath string) *Daemon {
	return &Daemon{
		binaryPath: binaryPath,
		dataPath:   dataPath,
		socketPath: socketPath,
	}
}

func (d *Daemon) Start() error {
	if d.IsRunning() {
		return nil
	}

	_ = os.Remove(d.socketPath)

	d.cmd = exec.Command(
		d.binaryPath,
		"--config", d.dataPath,
		"daemon",
		"--socket", d.socketPath,
		"--send-read-receipts",
	)

	d.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := d.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			_ = d.Stop() //nolint:errcheck
			return fmt.Errorf("daemon failed to start within timeout")
		case <-ticker.C:
			if _, err := os.Stat(d.socketPath); err == nil {
				return nil
			}
		}
	}
}

func (d *Daemon) Stop() error {
	if d.cmd == nil || d.cmd.Process == nil {
		return nil
	}

	if err := d.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- d.cmd.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		_ = d.cmd.Process.Kill()
	case <-done:
	}

	_ = os.Remove(d.socketPath)
	return nil
}

func (d *Daemon) IsRunning() bool {
	if _, err := os.Stat(d.socketPath); os.IsNotExist(err) {
		return false
	}

	client := NewClient(d.socketPath)
	_, err := client.Call("listAccounts", nil)
	return err == nil
}
