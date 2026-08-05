// Small helpers shared across commands.
package main

import (
	"fmt"
	"os"
	"os/exec"
)

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func getModulePath() (string, error) {
	// Try to read from go.mod
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "", err
	}

	// Parse module line
	lines := string(data)
	for _, line := range splitLines(lines) {
		if len(line) > 7 && line[:7] == "module " {
			return line[7:], nil
		}
	}

	return "", fmt.Errorf("module directive not found in go.mod")
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
