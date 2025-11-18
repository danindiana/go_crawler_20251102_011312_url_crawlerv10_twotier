package utils

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// NormalizeParsedURL normalizes a parsed URL
func NormalizeParsedURL(u *url.URL) string {
	u.Fragment = ""
	u.RawQuery = ""
	return strings.ToLower(u.String())
}

// IsDocumentURL checks if a URL points to a document
func IsDocumentURL(docURL string, extensions []string) bool {
	lowerURL := strings.ToLower(docURL)
	for _, ext := range extensions {
		if strings.HasSuffix(lowerURL, ext) ||
			strings.Contains(lowerURL, ext+"?") ||
			strings.Contains(lowerURL, ext+"&") {
			return true
		}
	}
	return false
}

// ExtractFilename extracts a filename from a URL and headers
func ExtractFilename(docURL string, headers http.Header) string {
	if cd := headers.Get("Content-Disposition"); cd != "" {
		if strings.HasPrefix(cd, "attachment; filename=") {
			filename := strings.TrimPrefix(cd, "attachment; filename=")
			filename = strings.Trim(filename, `"`)
			if filename != "" {
				return SanitizeFilename(filename)
			}
		}
	}

	segments := strings.Split(docURL, "/")
	filename := segments[len(segments)-1]

	if idx := strings.Index(filename, "?"); idx != -1 {
		filename = filename[:idx]
	}

	if filename == "" || !strings.Contains(filename, ".") {
		filename = fmt.Sprintf("download_%d", time.Now().UnixNano())
	}

	return SanitizeFilename(filename)
}

// SanitizeFilename removes invalid characters from a filename
func SanitizeFilename(name string) string {
	for _, ch := range []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|", "\x00"} {
		name = strings.ReplaceAll(name, ch, "_")
	}
	if len(name) > 200 {
		ext := filepath.Ext(name)
		name = name[:200-len(ext)] + ext
	}
	return name
}

// FormatBytes formats bytes into a human-readable string
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatMemory formats memory statistics into a human-readable string
func FormatMemory(m runtime.MemStats) string {
	return fmt.Sprintf("Alloc: %dMB, Sys: %dMB",
		m.Alloc/1024/1024, m.Sys/1024/1024)
}

// GetMemStats returns current memory statistics
func GetMemStats() runtime.MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m
}
