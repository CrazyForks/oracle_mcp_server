package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuditorLogWhitelistApproval(t *testing.T) {
	t.Parallel()

	logPath := filepath.Join(t.TempDir(), "audit.log")
	auditor, err := NewAuditor(logPath)
	if err != nil {
		t.Fatalf("NewAuditor returned error: %v", err)
	}
	defer auditor.Close()

	headerLine := "create or replace procedure demo"
	auditor.Log("select 1 from dual", []string{"create"}, ApprovalWhitelist, "SUCCESS", "play", &LogOptions{
		HeaderLine: &headerLine,
	})

	matches, err := filepath.Glob(filepath.Join(filepath.Dir(logPath), "audit_*.log"))
	if err != nil {
		t.Fatalf("Glob returned error: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 audit log file, got %d", len(matches))
	}

	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"AUDIT_APPROVED=whitelist",
		"HEADER_LINE=create or replace procedure demo",
		"AUDIT_CONNECTION=play",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("audit log missing %q:\n%s", want, content)
		}
	}
}
