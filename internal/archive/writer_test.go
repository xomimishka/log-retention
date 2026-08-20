package archive

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return filepath.ToSlash(p)
}

func TestArchiveFilesBasic(t *testing.T) {
	dir := t.TempDir()
	srcPath := writeFile(t, dir, "app.log", "hello world")
	info, _ := os.Stat(filepath.FromSlash(srcPath))

	target := filepath.ToSlash(filepath.Join(dir, "archive.zip"))
	sources := []SourceFile{{
		Path:      srcPath,
		EntryName: "app.log",
		Size:      info.Size(),
		ModTime:   info.ModTime(),
	}}

	gotPath, results, err := ArchiveFiles(target, sources, Options{Level: 9})
	if err != nil {
		t.Fatalf("ArchiveFiles failed: %v", err)
	}
	if gotPath != target {
		t.Errorf("path = %q, want %q", gotPath, target)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Name != "app.log" {
		t.Errorf("entry name = %q, want %q", results[0].Name, "app.log")
	}
	if results[0].UncompressedSize != info.Size() {
		t.Errorf("size = %d, want %d", results[0].UncompressedSize, info.Size())
	}

	reader, err := zip.OpenReader(filepath.FromSlash(target))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if len(reader.File) != 1 {
		t.Fatalf("archive has %d entries, want 1", len(reader.File))
	}
	if reader.File[0].Name != "app.log" {
		t.Errorf("archive entry = %q, want %q", reader.File[0].Name, "app.log")
	}
}

func TestArchiveFilesMultiple(t *testing.T) {
	dir := t.TempDir()
	src1 := writeFile(t, dir, "a.log", "aaa")
	src2 := writeFile(t, dir, "b.log", "bbb")

	info1, _ := os.Stat(filepath.FromSlash(src1))
	info2, _ := os.Stat(filepath.FromSlash(src2))

	target := filepath.ToSlash(filepath.Join(dir, "archive.zip"))
	sources := []SourceFile{
		{Path: src1, EntryName: "a.log", Size: info1.Size(), ModTime: info1.ModTime()},
		{Path: src2, EntryName: "b.log", Size: info2.Size(), ModTime: info2.ModTime()},
	}

	_, results, err := ArchiveFiles(target, sources, Options{Level: 6})
	if err != nil {
		t.Fatalf("ArchiveFiles failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
}

func TestArchiveFilesWithFolder(t *testing.T) {
	dir := t.TempDir()
	srcPath := writeFile(t, dir, "app.log", "hello")
	info, _ := os.Stat(filepath.FromSlash(srcPath))

	target := filepath.ToSlash(filepath.Join(dir, "archive.zip"))
	sources := []SourceFile{{
		Path:      srcPath,
		EntryName: "app.log",
		Size:      info.Size(),
		ModTime:   info.ModTime(),
	}}

	_, results, err := ArchiveFiles(target, sources, Options{
		Level:           9,
		FolderInArchive: "2026-08-19",
	})
	if err != nil {
		t.Fatalf("ArchiveFiles failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	want := "2026-08-19/app.log"
	if results[0].Name != want {
		t.Errorf("entry name = %q, want %q", results[0].Name, want)
	}
}

func TestArchiveFilesMergeSameDay(t *testing.T) {
	dir := t.TempDir()

	src1 := writeFile(t, dir, "a.log", "aaa")
	info1, _ := os.Stat(filepath.FromSlash(src1))
	target := filepath.ToSlash(filepath.Join(dir, "archive.zip"))

	_, _, err := ArchiveFiles(target, []SourceFile{{
		Path:      src1,
		EntryName: "a.log",
		Size:      info1.Size(),
		ModTime:   info1.ModTime(),
	}}, Options{Level: 9})
	if err != nil {
		t.Fatalf("first archive failed: %v", err)
	}

	src2 := writeFile(t, dir, "b.log", "bbb")
	info2, _ := os.Stat(filepath.FromSlash(src2))

	_, results, err := ArchiveFiles(target, []SourceFile{{
		Path:      src2,
		EntryName: "b.log",
		Size:      info2.Size(),
		ModTime:   info2.ModTime(),
	}}, Options{Level: 9, MergeSameDay: true})
	if err != nil {
		t.Fatalf("merge archive failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}

	reader, err := zip.OpenReader(filepath.FromSlash(target))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if len(reader.File) != 2 {
		t.Fatalf("archive has %d entries, want 2", len(reader.File))
	}
	names := map[string]bool{}
	for _, f := range reader.File {
		names[f.Name] = true
	}
	if !names["a.log"] || !names["b.log"] {
		t.Errorf("expected a.log and b.log, got %v", names)
	}
}

func TestArchiveFilesEntryCollision(t *testing.T) {
	dir := t.TempDir()

	src1 := writeFile(t, dir, "app1.log", "first")
	info1, _ := os.Stat(filepath.FromSlash(src1))
	target := filepath.ToSlash(filepath.Join(dir, "archive.zip"))

	_, _, err := ArchiveFiles(target, []SourceFile{{
		Path:      src1,
		EntryName: "app.log",
		Size:      info1.Size(),
		ModTime:   info1.ModTime(),
	}}, Options{Level: 9})
	if err != nil {
		t.Fatalf("first archive failed: %v", err)
	}

	src2 := writeFile(t, dir, "app2.log", "second")
	info2, _ := os.Stat(filepath.FromSlash(src2))

	_, results, err := ArchiveFiles(target, []SourceFile{{
		Path:      src2,
		EntryName: "app.log",
		Size:      info2.Size(),
		ModTime:   info2.ModTime(),
	}}, Options{Level: 9, MergeSameDay: true})
	if err != nil {
		t.Fatalf("merge archive failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Name != "app-2.log" {
		t.Errorf("entry name = %q, want %q", results[0].Name, "app-2.log")
	}
}

func TestArchiveFilesStale(t *testing.T) {
	dir := t.TempDir()
	srcPath := writeFile(t, dir, "app.log", "hello")
	info, _ := os.Stat(filepath.FromSlash(srcPath))

	target := filepath.ToSlash(filepath.Join(dir, "archive.zip"))

	sources := []SourceFile{{
		Path:      srcPath,
		EntryName: "app.log",
		Size:      info.Size() + 100,
		ModTime:   info.ModTime(),
	}}

	_, _, err := ArchiveFiles(target, sources, Options{Level: 9})
	if err == nil {
		t.Fatal("expected stale error")
	}
	if _, ok := err.(*StaleError); !ok {
		t.Logf("error type: %T, value: %v", err, err)
	}
}

func TestArchiveFilesEmptySources(t *testing.T) {
	dir := t.TempDir()
	target := filepath.ToSlash(filepath.Join(dir, "archive.zip"))

	_, _, err := ArchiveFiles(target, []SourceFile{}, Options{Level: 9})
	if err == nil {
		t.Fatal("expected error for empty sources")
	}
}

func TestArchiveFilesInvalidLevel(t *testing.T) {
	dir := t.TempDir()
	srcPath := writeFile(t, dir, "app.log", "hello")
	info, _ := os.Stat(filepath.FromSlash(srcPath))
	target := filepath.ToSlash(filepath.Join(dir, "archive.zip"))

	sources := []SourceFile{{
		Path:      srcPath,
		EntryName: "app.log",
		Size:      info.Size(),
		ModTime:   info.ModTime(),
	}}

	_, _, err := ArchiveFiles(target, sources, Options{Level: 10})
	if err == nil {
		t.Fatal("expected error for invalid level")
	}
}

func TestArchiveFilesCompressionLevels(t *testing.T) {
	dir := t.TempDir()
	content := "some repetitive content for compression test"
	srcPath := writeFile(t, dir, "app.log", content)
	info, _ := os.Stat(filepath.FromSlash(srcPath))

	for _, level := range []int{0, 1, 5, 9} {
		target := filepath.ToSlash(filepath.Join(dir, "archive-level"+string(rune('0'+level))+".zip"))
		sources := []SourceFile{{
			Path:      srcPath,
			EntryName: "app.log",
			Size:      info.Size(),
			ModTime:   info.ModTime(),
		}}
		_, _, err := ArchiveFiles(target, sources, Options{Level: level})
		if err != nil {
			t.Errorf("level %d: ArchiveFiles failed: %v", level, err)
		}
	}
}

func TestArchiveFilesCreatesDestDir(t *testing.T) {
	dir := t.TempDir()
	srcPath := writeFile(t, dir, "app.log", "hello")
	info, _ := os.Stat(filepath.FromSlash(srcPath))

	target := filepath.ToSlash(filepath.Join(dir, "sub", "deep", "archive.zip"))
	sources := []SourceFile{{
		Path:      srcPath,
		EntryName: "app.log",
		Size:      info.Size(),
		ModTime:   info.ModTime(),
	}}

	_, _, err := ArchiveFiles(target, sources, Options{Level: 9})
	if err != nil {
		t.Fatalf("ArchiveFiles failed: %v", err)
	}
	if !fileExists(target) {
		t.Error("archive not created in nested dir")
	}
}

func TestArchiveFilesModTimePreserved(t *testing.T) {
	dir := t.TempDir()
	srcPath := writeFile(t, dir, "app.log", "hello")
	info, _ := os.Stat(filepath.FromSlash(srcPath))

	target := filepath.ToSlash(filepath.Join(dir, "archive.zip"))
	sources := []SourceFile{{
		Path:      srcPath,
		EntryName: "app.log",
		Size:      info.Size(),
		ModTime:   info.ModTime(),
	}}

	_, _, err := ArchiveFiles(target, sources, Options{Level: 9})
	if err != nil {
		t.Fatal(err)
	}

	reader, err := zip.OpenReader(filepath.FromSlash(target))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	entry := reader.File[0]
	// Zip хранит время с точностью до 2 секунд (DOS-формат),
	// поэтому сравниваем с допуском.
	diff := entry.Modified.Sub(info.ModTime())
	if diff < 0 {
		diff = -diff
	}
	if diff > 2*time.Second {
		t.Errorf("mod time = %v, want %v (diff %v)", entry.Modified, info.ModTime(), diff)
	}
}
