package expand

import (
	"path/filepath"
	"runtime"
	"strings"
)

// проверяет, что маска корректна для filepath.Match
func ValidateMask(mask string) error {
	_, err := filepath.Match(mask, "")
	return err
}

// проверяет, соответствует ли имя файла маске
func Match(mask, name string) (bool, error) {
	if runtime.GOOS == "windows" {
		mask = strings.ToLower(mask)
		name = strings.ToLower(name)
	}
	return filepath.Match(mask, name)
}
