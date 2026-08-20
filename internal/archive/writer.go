package archive

import (
	"archive/zip"
	"compress/flate"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	"example/log-retention/internal/atomicfile"
)

type SourceFile struct {
	Path      string
	EntryName string
	Size      int64
	ModTime   time.Time
}

type Options struct {
	Level           int
	MergeSameDay    bool
	FolderInArchive string
}

type EntryResult struct {
	Name             string
	UncompressedSize int64
	CRC32            uint32
}

type StaleError struct {
	Path   string
	Reason string
}

func (e *StaleError) Error() string {
	return fmt.Sprintf("stale file %s: %s", e.Path, e.Reason)
}

func ArchiveFiles(target string, sources []SourceFile, opts Options) (string, []EntryResult, error) {
	if len(sources) == 0 {
		return "", nil, fmt.Errorf("no source files to archive")
	}
	if opts.Level < 0 || opts.Level > 9 {
		return "", nil, fmt.Errorf("invalid compression level %d, must be 0-9", opts.Level)
	}

	destDir := filepath.Dir(filepath.FromSlash(target))
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("create dest dir: %w", err)
	}

	exists := fileExists(target)
	merge := opts.MergeSameDay && exists

	existingNames := make(map[string]bool)

	var copyReader *zip.ReadCloser
	var tempCopyPath string
	if merge {
		existingReader, err := zip.OpenReader(filepath.FromSlash(target))
		if err != nil {
			return "", nil, fmt.Errorf("open existing archive: %w", err)
		}

		for _, f := range existingReader.File {
			existingNames[f.Name] = true
		}

		tmpCopy, err := os.CreateTemp(destDir, ".copy-")
		if err != nil {
			existingReader.Close()
			return "", nil, fmt.Errorf("create temp copy: %w", err)
		}
		tempCopyPath = tmpCopy.Name()

		zw := zip.NewWriter(tmpCopy)
		for _, f := range existingReader.File {
			if err := copyEntry(zw, f); err != nil {
				zw.Close()
				tmpCopy.Close()
				existingReader.Close()
				os.Remove(tempCopyPath)
				return "", nil, fmt.Errorf("copy entry %q: %w", f.Name, err)
			}
		}
		zw.Close()
		tmpCopy.Close()
		existingReader.Close()

		copyReader, err = zip.OpenReader(tempCopyPath)
		if err != nil {
			os.Remove(tempCopyPath)
			return "", nil, fmt.Errorf("open temp copy: %w", err)
		}
	}

	defer func() {
		if copyReader != nil {
			copyReader.Close()
		}
		if tempCopyPath != "" {
			os.Remove(tempCopyPath)
		}
	}()

	var results []EntryResult

	err := atomicfile.WriteFunc(target, func(w io.Writer) error {
		zw := zip.NewWriter(w)
		zw.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
			return flate.NewWriter(out, opts.Level)
		})

		if copyReader != nil {
			for _, f := range copyReader.File {
				if err := copyEntry(zw, f); err != nil {
					return fmt.Errorf("copy entry %q: %w", f.Name, err)
				}
			}
		}

		for _, src := range sources {
			result, err := addEntry(zw, src, opts, existingNames)
			if err != nil {
				return err
			}
			results = append(results, result)
		}

		return zw.Close()
	})
	if err != nil {
		return "", nil, err
	}

	if err := VerifyEntries(target, results); err != nil {
		return "", nil, fmt.Errorf("verify archive: %w", err)
	}

	return target, results, nil
}

func copyEntry(zw *zip.Writer, f *zip.File) error {
	header := f.FileHeader
	writer, err := zw.CreateHeader(&header)
	if err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	_, err = io.Copy(writer, rc)
	return err
}

func addEntry(zw *zip.Writer, src SourceFile, opts Options, existingNames map[string]bool) (EntryResult, error) {
	fullName := path.Join(opts.FolderInArchive, src.EntryName)
	if fullName == "" {
		return EntryResult{}, fmt.Errorf("empty entry name")
	}

	fullName = ResolveEntryCollision(fullName, func(name string) bool {
		return existingNames[name]
	})
	existingNames[fullName] = true

	diskPath := filepath.FromSlash(src.Path)
	srcFile, err := os.Open(diskPath)
	if err != nil {
		return EntryResult{}, fmt.Errorf("open source %q: %w", src.Path, err)
	}
	defer srcFile.Close()

	info, err := srcFile.Stat()
	if err != nil {
		return EntryResult{}, fmt.Errorf("stat source %q: %w", src.Path, err)
	}
	if info.Size() != src.Size {
		return EntryResult{}, &StaleError{
			Path:   src.Path,
			Reason: fmt.Sprintf("size changed: got %d, want %d", info.Size(), src.Size),
		}
	}
	if !info.ModTime().UTC().Equal(src.ModTime.UTC()) {
		return EntryResult{}, &StaleError{
			Path: src.Path,
			Reason: fmt.Sprintf("mod time changed: got %s, want %s",
				info.ModTime().UTC().Format(time.RFC3339),
				src.ModTime.UTC().Format(time.RFC3339)),
		}
	}

	header := &zip.FileHeader{
		Name:     fullName,
		Method:   zip.Deflate,
		Modified: src.ModTime.UTC(),
	}
	header.SetMode(0644)

	writer, err := zw.CreateHeader(header)
	if err != nil {
		return EntryResult{}, fmt.Errorf("create entry %q: %w", fullName, err)
	}

	hasher := crc32.NewIEEE()
	tee := io.TeeReader(srcFile, hasher)
	written, err := io.Copy(writer, tee)
	if err != nil {
		return EntryResult{}, fmt.Errorf("write entry %q: %w", fullName, err)
	}
	if written != info.Size() {
		return EntryResult{}, fmt.Errorf("write entry %q: wrote %d bytes, want %d",
			fullName, written, info.Size())
	}

	return EntryResult{
		Name:             fullName,
		UncompressedSize: written,
		CRC32:            hasher.Sum32(),
	}, nil
}

func VerifyEntries(archivePath string, expected []EntryResult) error {
	reader, err := zip.OpenReader(filepath.FromSlash(archivePath))
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer reader.Close()

	byName := make(map[string]*zip.File, len(reader.File))
	for _, f := range reader.File {
		byName[f.Name] = f
	}

	for _, exp := range expected {
		f, ok := byName[exp.Name]
		if !ok {
			return fmt.Errorf("entry %q not found in archive", exp.Name)
		}
		if f.UncompressedSize64 != uint64(exp.UncompressedSize) {
			return fmt.Errorf("entry %q: size mismatch: got %d, want %d",
				exp.Name, f.UncompressedSize64, exp.UncompressedSize)
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open entry %q: %w", exp.Name, err)
		}
		hasher := crc32.NewIEEE()
		if _, err := io.Copy(hasher, rc); err != nil {
			rc.Close()
			return fmt.Errorf("read entry %q: %w", exp.Name, err)
		}
		rc.Close()
		if hasher.Sum32() != exp.CRC32 {
			return fmt.Errorf("entry %q: CRC32 mismatch: got %d, want %d",
				exp.Name, hasher.Sum32(), exp.CRC32)
		}
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(filepath.FromSlash(path))
	return err == nil
}
