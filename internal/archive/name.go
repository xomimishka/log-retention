package archive

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

var invalidWindowsChars = []rune{'<', '>', ':', '"', '/', '\\', '|', '?', '*'}

var reservedWindowsNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
	"COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
	"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

func ResolveName(template, group string, date time.Time, host, policy string) string {
	result := template
	result = strings.ReplaceAll(result, "{group}", group)
	result = strings.ReplaceAll(result, "{date}", date.Format("2006-01-02"))
	result = strings.ReplaceAll(result, "{host}", host)
	result = strings.ReplaceAll(result, "{policy}", policy)
	return result
}

func SanitizeFileName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if isInvalidChar(r) {
			b.WriteRune('_')
		} else {
			b.WriteRune(r)
		}
	}
	sanitized := b.String()

	base := sanitized
	if idx := strings.IndexByte(sanitized, '.'); idx > 0 {
		base = sanitized[:idx]
	}
	if reservedWindowsNames[strings.ToUpper(base)] {
		sanitized = "_" + sanitized
	}

	if sanitized == "" {
		sanitized = "_"
	}

	return sanitized
}

func isInvalidChar(r rune) bool {
	if r < 32 {
		return true
	}
	for _, c := range invalidWindowsChars {
		if r == c {
			return true
		}
	}
	return false
}

func ResolveCollision(destDir, name string, mergeSameDay bool, fileExists func(string) bool) string {
	target := filepath.ToSlash(filepath.Join(destDir, name))

	if mergeSameDay && fileExists(target) {
		return target
	}

	if !fileExists(target) {
		return target
	}

	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)

	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d%s", base, i, ext)
		candidatePath := filepath.ToSlash(filepath.Join(destDir, candidate))
		if !fileExists(candidatePath) {
			return candidatePath
		}
	}
}

func ResolveEntryCollision(name string, entryExists func(string) bool) string {
	if !entryExists(name) {
		return name
	}

	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)

	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d%s", base, i, ext)
		if !entryExists(candidate) {
			return candidate
		}
	}
}
