package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDecimalPorts(t *testing.T) {
	for text, expected := range map[string]int{"08000": 8000, "01000": 1000, "00080": 80} {
		value, err := parsePort(text)
		if err != nil || value != expected {
			t.Fatalf("%s: %d %v", text, value, err)
		}
	}
	for _, text := range []string{"0", "65536", "abc", "-1"} {
		if _, err := parsePort(text); err == nil {
			t.Fatal("invalid port accepted")
		}
	}
}

func TestVersionAndHealthcheckDoNotCreateFiles(t *testing.T) {
	output := filepath.Join(t.TempDir(), "sublink")
	if runtime.GOOS == "windows" {
		output += ".exe"
	}
	build := exec.Command("go", "build", "-o", output, ".")
	if result, err := build.CombinedOutput(); err != nil {
		t.Fatalf("%s: %v", result, err)
	}
	directory := t.TempDir()
	command := exec.Command(output, "--version")
	command.Dir = directory
	result, err := command.CombinedOutput()
	if err != nil || strings.TrimSpace(string(result)) != "2.1.2" {
		t.Fatalf("%s: %v", result, err)
	}
	command = exec.Command(output, "healthcheck")
	command.Dir = directory
	if err := command.Run(); err == nil {
		t.Fatal("health probe accepted missing configuration")
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 0 {
		t.Fatal("read-only command created application data")
	}
}
