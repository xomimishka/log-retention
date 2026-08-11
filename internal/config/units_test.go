package config

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		in      string
		want    time.Duration
		wantErr bool
	}{
		// корректные значения
		{"24h", 24 * time.Hour, false},
		{"30d", 30 * 24 * time.Hour, false},
		{"15m", 15 * time.Minute, false},
		{"45s", 45 * time.Second, false},
		{"1d", 24 * time.Hour, false},
		{"0s", 0, false},

		// ошибки
		{"", 0, true},        // пусто
		{"24", 0, true},      // нет суффикса
		{"-5h", 0, true},     // отрицательное
		{"24x", 0, true},     // неизвестный суффикс
		{"24hours", 0, true}, // неизвестный суффикс
		{"h", 0, true},       // нет числа
		{"1.5h", 0, true},    // не целое
	}

	for _, tt := range tests {
		got, err := ParseDuration(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseDuration(%q) = %v, want error", tt.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseDuration(%q) unexpected error: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseDuration(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		// байты
		{"100", 100, false},
		{"100B", 100, false},
		{"0", 0, false},

		// десятичные
		{"1KB", 1000, false},
		{"1MB", 1000 * 1000, false},
		{"1GB", 1000 * 1000 * 1000, false},
		{"1TB", 1000 * 1000 * 1000 * 1000, false},

		// двоичные
		{"1KiB", 1024, false},
		{"1MiB", 1024 * 1024, false},
		{"1GiB", 1024 * 1024 * 1024, false},
		{"1TiB", 1024 * 1024 * 1024 * 1024, false},

		// ошибки
		{"", 0, true},      // пусто
		{"-5MiB", 0, true}, // отрицательное
		{"10Mb", 0, true},  // Mb недопустим
		{"10mib", 0, true}, // mib недопустим
		{"MiB", 0, true},   // нет числа
		{"1.5MB", 0, true}, // не целое
		{"1XB", 0, true},   // неизвестный суффикс
	}

	for _, tt := range tests {
		got, err := ParseSize(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseSize(%q) = %v, want error", tt.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseSize(%q) unexpected error: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseSize(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
