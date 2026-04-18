package reviewwhitelist

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreAddAndContains(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "whitelist.json")
	store := NewStore(path)

	found, err := store.Contains("play", "create or replace procedure demo")
	if err != nil {
		t.Fatalf("Contains returned error: %v", err)
	}
	if found {
		t.Fatal("Contains unexpectedly returned true before Add")
	}

	if err := store.Add("play", "create or replace procedure demo"); err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	if err := store.Add("play", "create or replace procedure demo"); err != nil {
		t.Fatalf("Add duplicate returned error: %v", err)
	}

	found, err = store.Contains("play", "create or replace procedure demo")
	if err != nil {
		t.Fatalf("Contains returned error after Add: %v", err)
	}
	if !found {
		t.Fatal("Contains returned false after Add")
	}

	found, err = store.Contains("play", "create or replace procedure other_demo")
	if err != nil {
		t.Fatalf("Contains returned error for different header: %v", err)
	}
	if found {
		t.Fatal("Contains unexpectedly matched a different header")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	want := "[\n  {\n    \"connection\": \"play\",\n    \"header_line\": \"create or replace procedure demo\"\n  }\n]\n"
	if string(data) != want {
		t.Fatalf("unexpected whitelist content:\n%s", string(data))
	}
}
