package clusters

import "strings"

func formatName(format string, replacements map[string]string) string {
	// Use default format if an empty format was passed
	if format == "" {
		format = defaultFormat
	}

	for old, replacement := range replacements {
		format = strings.ReplaceAll(format, old, replacement)
	}

	return format
}
