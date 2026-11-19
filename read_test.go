package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadConfigJSONValidatesAndDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	contents := []byte(`{"Port":"06969","EnableRateLimiting":true,"RateLimitAfter":3}`)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}

	settings, err := ReadConfigJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := settings.Port, "6969"; got != want {
		t.Fatalf("Port=%q, want %q", got, want)
	}
	if got, want := settings.RateLimitWindowSeconds, 60; got != want {
		t.Fatalf("RateLimitWindowSeconds=%d, want %d", got, want)
	}
}

func TestReadConfigJSONRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	contents := []byte(`{"Port":"6969","EnableWebFireWall":true}`)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := ReadConfigJSON(path); err == nil {
		t.Fatal("configuration with an unsupported field was accepted")
	}
}
