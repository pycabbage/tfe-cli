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
		t.Fatalf("行数: got %d, want 4", len(lines))
	}
	if !strings.Contains(lines[0], "ID") || !strings.Contains(lines[0], "Name") {
		t.Errorf("ヘッダー行に ID/Name が含まれていない: %q", lines[0])
	}
	if !strings.Contains(lines[1], "--") {
		t.Errorf("区切り行が正しくない: %q", lines[1])
	}
	if !strings.Contains(lines[2], "Alice") {
		t.Errorf("1行目のデータが正しくない: %q", lines[2])
	}
	if !strings.Contains(lines[3], "Bob") {
		t.Errorf("2行目のデータが正しくない: %q", lines[3])
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
	// ヘッダー行・区切り行・データ行はすべて同じ表示幅になる（パディングされる）
	if len(lines[0]) != len(lines[2]) {
		t.Errorf("ヘッダー行とデータ行の幅が異なる: header=%d, data=%d", len(lines[0]), len(lines[2]))
	}
}

func TestPrintTable_EmptyRows(t *testing.T) {
	headers := []string{"ID", "Name"}
	got := captureStdout(t, func() {
		PrintTable(headers, nil)
	})

	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("空行時は2行（ヘッダー+区切り）のみ期待: got %d行", len(lines))
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
		t.Fatalf("行数: got %d, want 3", len(lines))
	}

	// 各行で値が含まれていること
	if !strings.Contains(lines[0], "alice") {
		t.Errorf("1行目に 'alice' が含まれていない: %q", lines[0])
	}
	if !strings.Contains(lines[1], "alice@example.com") {
		t.Errorf("2行目に email が含まれていない: %q", lines[1])
	}

	// キーが揃っていること（"Username" と "Email" は "Two-Factor Auth" に合わせてパディング）
	usernameCol := strings.Index(lines[0], "alice")
	emailCol := strings.Index(lines[1], "alice@example.com")
	if usernameCol != emailCol {
		t.Errorf("値の開始位置がずれている: Username=%d, Email=%d", usernameCol, emailCol)
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
		t.Errorf("出力に Key/Value が含まれていない: %q", got)
	}
}
