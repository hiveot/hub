package utils

import (
	"fmt"
	"time"
)

// Age converts the given time to the current age format h m s
func Age(t time.Time) string {
	dur := int(time.Now().Sub(t).Round(time.Second).Seconds())
	days := dur / (24 * 3600)
	if days >= 1 {
		dur -= days * (24 * 3600)
	}
	hours := dur / 3600
	dur -= hours * 3600
	minutes := dur / 60
	sec := dur - minutes*60
	if days > 30 {
		return fmt.Sprintf("%dd, %dh", days, hours)
	} else if days > 0 {
		return fmt.Sprintf("%dd, %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, sec)
	}
	return fmt.Sprintf("%dm %ds", minutes, sec)
}
