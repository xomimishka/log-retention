package config

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

func ParseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("duration: empty value")
	}
	if strings.HasPrefix(s, "-") {
		return 0, fmt.Errorf("duration %q: negative value", s)
	}

	if s == "0" {
		return 0, nil
	}

	numStr, suffix := splitNumber(s)
	if numStr == "" {
		return 0, fmt.Errorf("duration %q: missing number", s)
	}
	if suffix == "" {
		return 0, fmt.Errorf("duration %q: missing suffix", s)
	}

	var mult time.Duration
	switch suffix {
	case "s":
		mult = time.Second
	case "m":
		mult = time.Minute
	case "h":
		mult = time.Hour
	case "d":
		mult = 24 * time.Hour
	default:
		return 0, fmt.Errorf("duration %q: unknown duration suffix %q", s, suffix)
	}

	n, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("duration %q: %w", s, err)
	}
	if n > math.MaxInt64/int64(mult) {
		return 0, fmt.Errorf("duration %q: value too large", s)
	}
	return time.Duration(n) * mult, nil
}

func ParseSize(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("size: empty value")
	}
	if strings.HasPrefix(s, "-") {
		return 0, fmt.Errorf("size %q: negative value", s)
	}

	// ноль разрешён, чтобы min_age: 0 можно было записать
	if s == "0" {
		return 0, nil
	}

	numStr, suffix := splitNumber(s)
	if numStr == "" {
		return 0, fmt.Errorf("size %q: missing number", s)
	}

	var mult int64
	switch suffix {
	case "", "B":
		mult = 1
	case "KB":
		mult = 1000
	case "MB":
		mult = 1000 * 1000
	case "GB":
		mult = 1000 * 1000 * 1000
	case "TB":
		mult = 1000 * 1000 * 1000 * 1000
	case "KiB":
		mult = 1 << 10
	case "MiB":
		mult = 1 << 20
	case "GiB":
		mult = 1 << 30
	case "TiB":
		mult = 1 << 40
	default:
		return 0, fmt.Errorf("size %q: unknown size suffix %q", s, suffix)
	}

	n, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("size %q: %w", s, err)
	}
	if n > math.MaxInt64/mult {
		return 0, fmt.Errorf("size %q: value too large", s)
	}
	return n * mult, nil
}

func splitNumber(s string) (num, suffix string) {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return s[:i], s[i:]
}
