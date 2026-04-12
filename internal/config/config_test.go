package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessConnectionsEncryptsPlaintext(t *testing.T) {
	t.Parallel()

	plain, encrypted, changed, err := processConnections(map[string]string{
		"play": "play/pass@//host:1521/demo",
	})
	if err != nil {
		t.Fatalf("processConnections returned error: %v", err)
	}
	if !changed {
		t.Fatalf("processConnections changed = false, want true")
	}
	if plain["play"] != "play/pass@//host:1521/demo" {
		t.Fatalf("plain connection = %q", plain["play"])
	}
	if !isEncryptedConnectionValue(encrypted["play"]) {
		t.Fatalf("encrypted connection = %q, want encrypted prefix", encrypted["play"])
	}
}

func TestProcessConnectionsDecryptsEncrypted(t *testing.T) {
	t.Parallel()

	ciphertext, err := encryptConnectionValue("play/pass@//host:1521/demo")
	if err != nil {
		t.Fatalf("encryptConnectionValue returned error: %v", err)
	}

	plain, encrypted, changed, err := processConnections(map[string]string{
		"play": ciphertext,
	})
	if err != nil {
		t.Fatalf("processConnections returned error: %v", err)
	}
	if changed {
		t.Fatalf("processConnections changed = true, want false")
	}
	if plain["play"] != "play/pass@//host:1521/demo" {
		t.Fatalf("plain connection = %q", plain["play"])
	}
	if encrypted["play"] != ciphertext {
		t.Fatalf("encrypted connection changed unexpectedly")
	}
}

func TestLoadFromFileEncryptsConnectionsOnDiskAndReturnsPlaintext(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `oracle:
  connections:
    play: "play/pass@//host:1521/demo"
security:
  danger_keyword_match: "whole_text"
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile returned error: %v", err)
	}
	if cfg.Oracle.Connections["play"] != "play/pass@//host:1521/demo" {
		t.Fatalf("loaded connection = %q", cfg.Oracle.Connections["play"])
	}

	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(updated), connectionPrefix) {
		t.Fatalf("updated config does not contain encrypted connection: %s", string(updated))
	}
	if strings.Contains(string(updated), "play/pass@//host:1521/demo") {
		t.Fatalf("updated config still contains plaintext connection: %s", string(updated))
	}
}
