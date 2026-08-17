package atomicfile

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteBytesSuccess(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "output.txt")

	content := []byte("hello world")
	if err := WriteBytes(target, content); err != nil {
		t.Fatalf("WriteBytes failed: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content = %q, want %q", got, content)
	}
}

func TestWriteBytesCreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "sub", "deep", "output.txt")

	if err := WriteBytes(target, []byte("data")); err != nil {
		t.Fatalf("WriteBytes failed: %v", err)
	}

	if _, err := os.Stat(target); err != nil {
		t.Fatalf("target not created: %v", err)
	}
}

func TestWriteBytesErrorLeavesTargetUntouched(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "output.txt")

	original := []byte("original content")
	if err := WriteBytes(target, original); err != nil {
		t.Fatalf("initial write failed: %v", err)
	}

	err := WriteFunc(target, func(w io.Writer) error {
		w.Write([]byte("partial"))
		return errors.New("simulated failure")
	})
	if err == nil {
		t.Fatal("expected error from WriteFunc")
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(got) != string(original) {
		t.Errorf("target was modified: got %q, want %q", got, original)
	}
}

func TestWriteBytesNoTempFileLeftOnError(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "output.txt")

	_ = WriteFunc(target, func(w io.Writer) error {
		w.Write([]byte("data"))
		return errors.New("fail")
	})

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp-") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

func TestWriteBytesOverwriteExisting(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "output.txt")

	if err := WriteBytes(target, []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := WriteBytes(target, []byte("second")); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "second" {
		t.Errorf("content = %q, want %q", got, "second")
	}
}
