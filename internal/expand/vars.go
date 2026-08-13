package expand

import (
	"fmt"
	"os"
	"strings"
)

func ExpandVars(s, policy, field string, defaults map[string]string) (string, error) {
	var b strings.Builder
	i := 0
	for i < len(s) {
		switch {
		case strings.HasPrefix(s[i:], "${"):
			end := strings.Index(s[i:], "}")
			if end == -1 {
				b.WriteByte(s[i])
				i++
				continue
			}
			name := s[i+2 : i+end]
			if !isValidVarName(name) {
				b.WriteByte(s[i])
				i++
				continue
			}
			val, err := lookupVar(name, policy, field, defaults)
			if err != nil {
				return "", err
			}
			b.WriteString(val)
			i += end + 1

		case s[i] == '%':
			end := strings.IndexByte(s[i+1:], '%')
			if end == -1 {
				b.WriteByte(s[i])
				i++
				continue
			}
			name := s[i+1 : i+1+end]
			if !isValidVarName(name) {
				b.WriteByte(s[i])
				i++
				continue
			}
			val, err := lookupVar(name, policy, field, defaults)
			if err != nil {
				return "", err
			}
			b.WriteString(val)
			i += end + 2

		default:
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String(), nil
}

func lookupVar(name, policy, field string, defaults map[string]string) (string, error) {
	def, ok := defaults[name]
	if !ok {
		return "", fmt.Errorf("policy %q field %q: unknown variable %q", policy, field, name)
	}
	if env := strings.TrimSpace(os.Getenv(name)); env != "" {
		return env, nil
	}
	if def != "" {
		return def, nil
	}
	return "", fmt.Errorf("policy %q field %q: variable %q has no value and no default", policy, field, name)
}

func isValidVarName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if !isLetter(r) && r != '_' {
				return false
			}
		} else {
			if !isLetter(r) && !isDigit(r) && r != '_' {
				return false
			}
		}
	}
	return true
}

func isLetter(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}
