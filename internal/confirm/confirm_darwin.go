//go:build darwin

// Package confirm provides Human-in-the-loop confirmation dialogs for macOS.
package confirm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ConfirmRequest contains the data for a confirmation dialog.
type ConfirmRequest struct {
	SQL                         string
	MatchedKeywords             []string
	DisplayKeywords             []string
	MatchedKeywordsForHighlight []string
	MatchedActions              []string
	StatementType               string
	IsDDL                       bool
	Connection                  string
	ConnectionIndex             int
	SourceLabel                 string
	WhitelistPath               string
	WhitelistConnection         string
	ReviewTriggerDetails        []string
	DangerKeywords              []string
}

// ConfirmResult reports how the review dialog was approved.
type ConfirmResult struct {
	Approved    bool
	AllowHeader bool
}

// Confirmer handles user confirmation dialogs on macOS.
type Confirmer struct{}

// NewConfirmer creates a new Confirmer instance.
func NewConfirmer() *Confirmer {
	return &Confirmer{}
}

// Confirm shows a confirmation dialog using JXA and returns the approval result.
func (c *Confirmer) Confirm(req *ConfirmRequest) (ConfirmResult, error) {
	sqlDir := os.TempDir()
	resultPath := filepath.Join(sqlDir, "oracle-mcp-confirm-result.txt")
	scriptPath := filepath.Join(sqlDir, "oracle-mcp-confirm-dialog.js")
	headerPath := filepath.Join(sqlDir, "oracle-mcp-confirm-header.txt")
	sqlPath := filepath.Join(sqlDir, "oracle-mcp-confirm-sql.txt")
	dangerKeywordsPath := filepath.Join(sqlDir, "oracle-mcp-confirm-danger-keywords.json")

	if err := os.WriteFile(headerPath, []byte(buildConfirmHeader(req)), 0600); err != nil {
		return ConfirmResult{}, fmt.Errorf("confirm: cannot write header temp file: %w", err)
	}
	defer os.Remove(headerPath)
	defer os.Remove(resultPath)

	if err := os.WriteFile(sqlPath, []byte(req.SQL), 0600); err != nil {
		return ConfirmResult{}, fmt.Errorf("confirm: cannot write sql temp file: %w", err)
	}
	defer os.Remove(sqlPath)

	dangerKeywordsJSON := "[]"
	if len(req.DangerKeywords) > 0 {
		quoted := make([]string, 0, len(req.DangerKeywords))
		for _, keyword := range req.DangerKeywords {
			escaped := strings.ReplaceAll(keyword, `\`, `\\`)
			escaped = strings.ReplaceAll(escaped, `"`, `\"`)
			quoted = append(quoted, `"`+escaped+`"`)
		}
		dangerKeywordsJSON = "[" + strings.Join(quoted, ",") + "]"
	}
	if err := os.WriteFile(dangerKeywordsPath, []byte(dangerKeywordsJSON), 0600); err != nil {
		return ConfirmResult{}, fmt.Errorf("confirm: cannot write danger keywords temp file: %w", err)
	}
	defer os.Remove(dangerKeywordsPath)

	if err := os.WriteFile(scriptPath, []byte(jxaScript), 0600); err != nil {
		return ConfirmResult{}, fmt.Errorf("confirm: cannot write script temp file: %w", err)
	}
	defer os.Remove(scriptPath)

	connectionArg := req.Connection
	if connectionArg == "" {
		connectionArg = "default"
	}

	cmd := exec.Command("osascript", "-l", "JavaScript", scriptPath,
		headerPath, sqlPath, resultPath, req.WhitelistPath, req.WhitelistConnection, dangerKeywordsPath, connectionArg)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 || strings.Contains(string(exitErr.Stderr), "-128") {
				return ConfirmResult{}, nil
			}
		}
		return ConfirmResult{}, fmt.Errorf("dialog error: %w", err)
	}

	switch strings.TrimSpace(string(output)) {
	case "1":
		return ConfirmResult{Approved: true}, nil
	case "allow_header":
		return ConfirmResult{Approved: true, AllowHeader: true}, nil
	default:
		return ConfirmResult{}, nil
	}
}

func buildConfirmHeader(req *ConfirmRequest) string {
	var line1 []string
	if req.Connection != "" {
		line1 = append(line1, "Database: "+req.Connection)
	}
	if len(req.MatchedActions) > 0 {
		line1 = append(line1, "Action: "+strings.Join(req.MatchedActions, ", "))
	} else if req.StatementType != "" {
		line1 = append(line1, "Action: "+req.StatementType)
	}
	displayKeywords := req.DisplayKeywords
	if len(displayKeywords) == 0 {
		displayKeywords = req.MatchedKeywords
	}
	if len(displayKeywords) > 0 {
		line1 = append(line1, "Keywords: "+strings.Join(displayKeywords, ", "))
	}
	if req.IsDDL {
		line1 = append(line1, "DDL (auto-committed)")
	}
	var out string
	if len(line1) > 0 {
		out = strings.Join(line1, "    |    ")
	}
	if req.SourceLabel != "" {
		if out != "" {
			out += "\n"
		}
		out += req.SourceLabel
	}
	if out == "" {
		return "Confirm SQL execution"
	}
	return out
}

// ShowError displays an error message dialog on macOS.
func (c *Confirmer) ShowError(title, message string) {
	script := fmt.Sprintf(`
		display dialog %q with title %q buttons {"OK"} default button "OK" with icon stop
	`, message, title)
	exec.Command("osascript", "-e", script).Run()
}

// ShowInfo displays an informational message dialog on macOS.
func (c *Confirmer) ShowInfo(title, message string) {
	script := fmt.Sprintf(`
		display dialog %q with title %q buttons {"OK"} default button "OK" with icon note
	`, message, title)
	exec.Command("osascript", "-e", script).Run()
}

// Available returns true on macOS.
func (c *Confirmer) Available() bool {
	return true
}

// PlatformName returns the platform name.
func (c *Confirmer) PlatformName() string {
	return "darwin"
}

// FormatConfirmationMessage formats the confirmation message for logging.
func FormatConfirmationMessage(req *ConfirmRequest) string {
	conn := req.Connection
	if conn == "" {
		conn = "default"
	}
	return fmt.Sprintf(
		"Connection=[%s] SQL=[%s] Keywords=[%s] Type=[%s] IsDDL=[%v]",
		conn,
		truncateSQL(req.SQL, 100),
		strings.Join(req.MatchedKeywords, ","),
		req.StatementType,
		req.IsDDL,
	)
}

func truncateSQL(sql string, maxLen int) string {
	sql = strings.ReplaceAll(sql, "\n", " ")
	sql = strings.ReplaceAll(sql, "\r", "")
	if len(sql) > maxLen {
		return sql[:maxLen] + "..."
	}
	return sql
}

const jxaScript = `
ObjC.import('AppKit');
ObjC.import('Foundation');

function run(argv) {
  const headerPath = argv[0];
  const sqlPath = argv[1];
  const resultPath = argv[2];
  const whitelistPath = argv[3];
  const whitelistConnection = argv[4] || "default";
  const dangerKeywordsPath = argv[5];
  const connectionName = argv[6] || "default";

  const app = Application.currentApplication();
  app.includeStandardAdditions = true;

  const header = readTextFile(headerPath) || "Confirm SQL execution";
  const sqlText = readTextFile(sqlPath) || "";
  const dangerKeywords = readJSONFile(dangerKeywordsPath, []);
  let keywordText = "";

  while (true) {
    const ui = buildAlert(connectionName, header, sqlText, keywordText);
    const response = ui.alert.runModal();
    keywordText = ObjC.unwrap(ui.keywordField.stringValue);

    if (response === 1000) {
      const trimmed = keywordText.trim();
      const validationError = validateKeyword(trimmed, dangerKeywords);
      if (validationError) {
        app.displayAlert("Allow Keyword", {message: validationError});
        continue;
      }
      if (!whitelistPath) {
        app.displayAlert("Allow Keyword", {message: "Whitelist path is unavailable."});
        continue;
      }
      try {
        saveWhitelistKeyword(whitelistPath, whitelistConnection, trimmed);
        app.displayAlert("Allow Keyword", {message: "Keyword added successfully."});
        keywordText = "";
      } catch (error) {
        app.displayAlert("Allow Keyword", {message: "Failed to update whitelist.json: " + error});
      }
      continue;
    }
    if (response === 1001) {
      writeResult(resultPath, "allow_header");
      return "allow_header";
    }
    if (response === 1002) {
      writeResult(resultPath, "1");
      return "1";
    }

    writeResult(resultPath, "0");
    return "0";
  }
}

function buildAlert(connectionName, header, sqlText, keywordText) {
  const alert = $.NSAlert.alloc.init;
  alert.messageText = "Confirm SQL — " + connectionName;
  alert.informativeText = header;
  alert.addButtonWithTitle("Allow Keyword");
  alert.addButtonWithTitle("Allow Header");
  alert.addButtonWithTitle("Execute");
  alert.addButtonWithTitle("Cancel");
  alert.alertStyle = $.NSAlertStyleWarning;

  const container = $.NSView.alloc.initWithFrame($.NSMakeRect(0, 0, 720, 420));

  const scrollView = $.NSScrollView.alloc.initWithFrame($.NSMakeRect(0, 70, 720, 350));
  scrollView.hasVerticalScroller = true;
  scrollView.hasHorizontalScroller = false;
  scrollView.borderType = $.NSBezelBorder;

  const textView = $.NSTextView.alloc.initWithFrame($.NSMakeRect(0, 0, 700, 350));
  textView.setEditable(false);
  textView.setSelectable(true);
  textView.setString($(sqlText));
  scrollView.documentView = textView;
  container.addSubview(scrollView);

  const label = $.NSTextField.labelWithString("Keyword:");
  label.setFrame($.NSMakeRect(0, 36, 100, 24));
  container.addSubview(label);

  const keywordField = $.NSTextField.alloc.initWithFrame($.NSMakeRect(80, 32, 640, 28));
  keywordField.setStringValue($(keywordText || ""));
  container.addSubview(keywordField);

  alert.accessoryView = container;
  return {alert: alert, keywordField: keywordField};
}

function validateKeyword(keyword, dangerKeywords) {
  if (!keyword) {
    return "Please enter a keyword first.";
  }
  if (!/^[A-Za-z0-9_]+$/.test(keyword)) {
    return "Keyword can contain only letters, numbers, and underscores.";
  }
  for (let i = 0; i < dangerKeywords.length; i++) {
    if (String(dangerKeywords[i]).toLowerCase() === keyword.toLowerCase()) {
      return "This keyword matches danger_keywords exactly and cannot be added.";
    }
  }
  return "";
}

function readTextFile(path) {
  if (!path) return "";
  const str = $.NSString.stringWithContentsOfFileEncodingError($(path), $.NSUTF8StringEncoding, null);
  return str ? ObjC.unwrap(str) : "";
}

function readJSONFile(path, fallbackValue) {
  const text = readTextFile(path).trim();
  if (!text) return fallbackValue;
  return JSON.parse(text);
}

function writeResult(path, value) {
  $(value).writeToFileAtomicallyEncodingError($(path), true, $.NSUTF8StringEncoding, null);
}

function saveWhitelistKeyword(path, connectionName, keyword) {
  let items = [];
  const raw = readTextFile(path).trim();
  if (raw) {
    const parsed = JSON.parse(raw);
    items = parsed.map(normalizeWhitelistEntry);
  }

  let entry = null;
  for (let i = 0; i < items.length; i++) {
    if (String(items[i].connection) === String(connectionName)) {
      entry = items[i];
      break;
    }
  }
  if (!entry) {
    entry = {connection: String(connectionName), head_line: [], "keyword:": []};
    items.push(entry);
  }

  const exists = entry["keyword:"].some(function (item) {
    return String(item).toLowerCase() === String(keyword).toLowerCase();
  });
  if (!exists) {
    entry["keyword:"].push(String(keyword));
  }

  const jsonText = JSON.stringify(items, null, 2) + "\n";
  $(jsonText).writeToFileAtomicallyEncodingError($(path), true, $.NSUTF8StringEncoding, null);
}

function normalizeWhitelistEntry(entry) {
  const normalized = {
    connection: String(entry.connection || ""),
    head_line: [],
    "keyword:": []
  };

  if (Array.isArray(entry.head_line)) {
    normalized.head_line = entry.head_line.map(String);
  } else if (typeof entry.header_line === "string") {
    normalized.head_line = [String(entry.header_line)];
  }

  if (Array.isArray(entry["keyword:"])) {
    normalized["keyword:"] = entry["keyword:"].map(String);
  }

  return normalized;
}
`
