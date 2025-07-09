package utils

import (
	"fmt"
	"time"
)

// FormatTime formats milliseconds into a human-readable duration string
func FormatTime(ms int) string {
	duration := time.Duration(ms) * time.Millisecond

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}
