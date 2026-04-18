package mcp

import "testing"

func TestFirstSQLLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		sql  string
		want string
	}{
		{name: "single line", sql: "select 1 from dual", want: "select 1 from dual"},
		{name: "multiline lf", sql: "create or replace procedure demo\nbegin\nnull;\nend;", want: "create or replace procedure demo"},
		{name: "multiline crlf", sql: "update demo\r\nset flag = 1", want: "update demo"},
		{name: "leading blank line", sql: "\nselect 1 from dual", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := firstSQLLine(tt.sql); got != tt.want {
				t.Fatalf("firstSQLLine(%q) = %q, want %q", tt.sql, got, tt.want)
			}
		})
	}
}
