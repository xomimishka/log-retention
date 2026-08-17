package atomicfile

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func WriteFunc(target string, fn func(w io.Writer) error) (err error) {
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".tmp-"+filepath.Base(target)+"-")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()

	success := false
	defer func() {
		if !success {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if err := fn(tmp); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	_ = os.Remove(target)

	if err := os.Rename(tmpName, target); err != nil {
		return fmt.Errorf("rename temp to target: %w", err)
	}

	success = true
	return nil
}

func WriteBytes(target string, data []byte) error {
	return WriteFunc(target, func(w io.Writer) error {
		_, err := w.Write(data)
		return err
	})
}
