package reviewwhitelist

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreAddAndContainsHeadLine(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "whitelist.json")
	store := NewStore(path)

	found, err := store.ContainsHeadLine("play", "create or replace procedure demo")
	if err != nil {
		t.Fatalf("ContainsHeadLine returned error: %v", err)
	}
	if found {
		t.Fatal("ContainsHeadLine unexpectedly returned true before AddHeadLine")
	}

	if err := store.AddHeadLine("play", "create or replace procedure demo"); err != nil {
		t.Fatalf("AddHeadLine returned error: %v", err)
	}
	if err := store.AddHeadLine("play", "create or replace procedure demo"); err != nil {
		t.Fatalf("AddHeadLine duplicate returned error: %v", err)
	}
	if err := store.AddKeyword("play", "created_at"); err != nil {
		t.Fatalf("AddKeyword returned error: %v", err)
	}

	found, err = store.ContainsHeadLine("play", "create or replace procedure demo")
	if err != nil {
		t.Fatalf("ContainsHeadLine returned error after AddHeadLine: %v", err)
	}
	if !found {
		t.Fatal("ContainsHeadLine returned false after AddHeadLine")
	}

	found, err = store.ContainsHeadLine("play", "create or replace procedure other_demo")
	if err != nil {
		t.Fatalf("ContainsHeadLine returned error for different header: %v", err)
	}
	if found {
		t.Fatal("ContainsHeadLine unexpectedly matched a different header")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	want := "[\n  {\n    \"connection\": \"play\",\n    \"head_line\": [\n      \"create or replace procedure demo\"\n    ],\n    \"keyword:\": [\n      \"created_at\"\n    ]\n  }\n]\n"
	if string(data) != want {
		t.Fatalf("unexpected whitelist content:\n%s", string(data))
	}
}

func TestStoreLoadsLegacyHeaderLineFormat(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "whitelist.json")
	legacy := "[\n  {\n    \"connection\": \"play\",\n    \"header_line\": \"CREATE OR REPLACE PROCEDURE al_test IS\"\n  }\n]\n"
	if err := os.WriteFile(path, []byte(legacy), 0644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	store := NewStore(path)
	found, err := store.ContainsHeadLine("play", "CREATE OR REPLACE PROCEDURE al_test IS")
	if err != nil {
		t.Fatalf("ContainsHeadLine returned error: %v", err)
	}
	if !found {
		t.Fatal("ContainsHeadLine did not match migrated legacy content")
	}

	if err := store.AddKeyword("play", "created_at"); err != nil {
		t.Fatalf("AddKeyword returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	want := "[\n  {\n    \"connection\": \"play\",\n    \"head_line\": [\n      \"CREATE OR REPLACE PROCEDURE al_test IS\"\n    ],\n    \"keyword:\": [\n      \"created_at\"\n    ]\n  }\n]\n"
	if string(data) != want {
		t.Fatalf("unexpected migrated whitelist content:\n%s", string(data))
	}
}
