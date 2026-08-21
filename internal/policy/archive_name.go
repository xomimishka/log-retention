package policy

import (
	"os"
	"strings"
	"time"

	"example/log-retention/internal/archive"
	"example/log-retention/internal/config"
)

func RenderArchiveTarget(p config.Policy, group string, now time.Time) string {
	dest := p.Archive.Dest
	if dest == "" {
		dest = "/tmp/archive"
	}
	name := p.Archive.Name
	if name == "" {
		name = "{group}-{date}.zip"
	}

	loc := time.UTC
	if p.Schedule.Timezone != "" {
		if l, err := time.LoadLocation(p.Schedule.Timezone); err == nil {
			loc = l
		}
	}

	substitutions := map[string]string{
		"{group}":  group,
		"{date}":   now.In(loc).Format("2006-01-02"),
		"{host}":   hostname(),
		"{policy}": p.Name,
	}

	renderedDest := renderTemplate(dest, substitutions)
	renderedName := renderTemplate(name, substitutions)

	sanitizedName := archive.SanitizeFileName(renderedName)

	target := renderedDest + "/" + sanitizedName
	return toSlashClean(target)
}

func renderTemplate(template string, substitutions map[string]string) string {
	result := template
	for placeholder, value := range substitutions {
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

func hostname() string {
	name, err := os.Hostname()
	if err != nil || name == "" {
		return "unknown"
	}
	return name
}

func toSlashClean(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")

	for strings.Contains(p, "//") {
		p = strings.ReplaceAll(p, "//", "/")
	}
	return p
}
