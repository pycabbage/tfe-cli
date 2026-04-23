package output

import (
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("pipe close: %v", err)
	}
	os.Stdout = orig
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	return string(out)
}

func TestPrintTable_Basic(t *testing.T) {
	headers := []string{"ID", "Name"}
	rows := [][]string{
		{"1", "Alice"},
		{"2", "Bob"},
	}
	got := captureStdout(t, func() {
		PrintTable(headers, rows)
	})

	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("row count: got %d, want 4", len(lines))
	}
	if !strings.Contains(lines[0], "ID") || !strings.Contains(lines[0], "Name") {
		t.Errorf("header row missing ID/Name: %q", lines[0])
	}
	if !strings.Contains(lines[1], "--") {
		t.Errorf("incorrect separator row: %q", lines[1])
	}
	if !strings.Contains(lines[2], "Alice") {
		t.Errorf("incorrect first data row: %q", lines[2])
	}
	if !strings.Contains(lines[3], "Bob") {
		t.Errorf("incorrect second data row: %q", lines[3])
	}
}

func TestPrintTable_CellWiderThanHeader(t *testing.T) {
	headers := []string{"ID"}
	rows := [][]string{
		{"very-long-id-value"},
	}
	got := captureStdout(t, func() {
		PrintTable(headers, rows)
	})

	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines[0]) != len(lines[2]) {
		t.Errorf("header and data row widths differ: header=%d, data=%d", len(lines[0]), len(lines[2]))
	}
}

func TestPrintTable_EmptyRows(t *testing.T) {
	headers := []string{"ID", "Name"}
	got := captureStdout(t, func() {
		PrintTable(headers, nil)
	})

	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (header + separator) for empty data: got %d lines", len(lines))
	}
}

func TestPrintKV_Basic(t *testing.T) {
	pairs := [][2]string{
		{"Username", "alice"},
		{"Email", "alice@example.com"},
		{"Two-Factor Auth", "enabled"},
	}
	got := captureStdout(t, func() {
		PrintKV(pairs)
	})

	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("row count: got %d, want 3", len(lines))
	}

	if !strings.Contains(lines[0], "alice") {
		t.Errorf("first row missing 'alice': %q", lines[0])
	}
	if !strings.Contains(lines[1], "alice@example.com") {
		t.Errorf("second row missing email: %q", lines[1])
	}

	usernameCol := strings.Index(lines[0], "alice")
	emailCol := strings.Index(lines[1], "alice@example.com")
	if usernameCol != emailCol {
		t.Errorf("value start position misaligned: Username=%d, Email=%d", usernameCol, emailCol)
	}
}

func TestPrintKV_SinglePair(t *testing.T) {
	pairs := [][2]string{
		{"Key", "Value"},
	}
	got := captureStdout(t, func() {
		PrintKV(pairs)
	})

	if !strings.Contains(got, "Key") || !strings.Contains(got, "Value") {
		t.Errorf("output missing Key/Value: %q", got)
	}
}
