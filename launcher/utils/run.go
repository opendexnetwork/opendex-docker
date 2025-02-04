package utils

import (
	"bytes"
	"context"
	"fmt"
	"github.com/opendexnetwork/opendex-docker/launcher/log"
	"os"
	"os/exec"
	"strings"
)

var logger = log.NewLogger("utils")

func Run(ctx context.Context, cmd *exec.Cmd) error {
	cmd.Env = append(cmd.Env, os.Environ()...)

	var buf bytes.Buffer

	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	done := make(chan error)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		// exited
		output := strings.TrimSpace(buf.String())
		if output != "" {
			output = "\n" + output
		}
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				logger.Errorf("[run] %s (exit %d)%s", cmd.String(), exitErr.ExitCode(), output)
			}
			return fmt.Errorf("[run] %s: %w", cmd.String(), err)
		} else {
			logger.Debugf("[run] %s%s", cmd.String(), output)
		}
	case <-ctx.Done():
		// cancelled
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("kill: %w", err)
		}
	}
	return nil
}

func Output(ctx context.Context, cmd *exec.Cmd) (string, error) {
	var buf bytes.Buffer

	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start: %w", err)
	}
	done := make(chan error)
	output := ""
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		// exited
		output += strings.TrimSpace(buf.String())
		if output != "" {
			output = "\n" + output
		}
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				logger.Errorf("[run] %s (exit %d)%s", cmd.String(), exitErr.ExitCode(), output)
			}
			return output, fmt.Errorf("[run] %s: %w", cmd.String(), err)
		} else {
			logger.Debugf("[run] %s%s", cmd.String(), output)
		}
	case <-ctx.Done():
		// cancelled
		if err := cmd.Process.Kill(); err != nil {
			return "", fmt.Errorf("kill: %w", err)
		}
	}
	return output, nil
}
