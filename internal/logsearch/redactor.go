package logsearch

import "regexp"

type redactor struct {
	patterns []*regexp.Regexp
}

func newRedactor() redactor {
	return redactor{patterns: []*regexp.Regexp{
		regexp.MustCompile(`(?i)(authorization\s*[:=]\s*)(bearer\s+)?[^\s,;]+`),
		regexp.MustCompile(`(?i)((?:access_?token|refresh_?token|password|passwd|secret|cookie|session)\s*["']?\s*[:=]\s*["']?)[^\s,"';}]+`),
		regexp.MustCompile(`(?i)((?:mobile|phone)\s*["']?\s*[:=]\s*["']?)\+?[0-9][0-9 -]{6,18}[0-9]`),
	}}
}

func (r redactor) Redact(line string) string {
	for _, pattern := range r.patterns {
		line = pattern.ReplaceAllString(line, `${1}[REDACTED]`)
	}
	return line
}
