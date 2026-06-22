// internal/utils/correlation.go

package utils

import (
    "crypto/rand"
    "encoding/hex"
    "time"
)

// GenerateCorrelationID creates a unique identifier for request tracing
func GenerateCorrelationID() string {
    bytes := make([]byte, 8)
    if _, err := rand.Read(bytes); err != nil {
        // Fallback to timestamp-based ID if crypto/rand fails
        return time.Now().UTC().Format("20060102150405") + "-fallback"
    }
    return hex.EncodeToString(bytes)
}

// TruncateString truncates a string to the specified length
func TruncateString(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return s[:maxLen] + "..."
}

// Ptr returns a pointer to a value (useful for optional fields)
func Ptr[T any](v T) *T {
    return &v
}