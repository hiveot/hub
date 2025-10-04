package utils

import (
	"fmt"
	"time"

	"github.com/araddon/dateparse"
)

// each of these formats can be updated
var WeekTimeFormat = "Mon 02, 15:04:05 MST"     // for the last 7 days
var YearTimeFormat = "2006-01-02, 15:04:05 MST" // full date-time
var VerboseTimeFormat = time.RFC1123            // full date-time

var MilliTimeFormat = "2006-01-02 15:04:05.000 MST" // millisecond
const rfc3339Milli = "2006-01-02T15:04:05.999Z"

// FormatAuto automatic date/time formatting based on age
//func FormatAuto(dateStr string) (formattedTime string) {
//	if dateStr == "" {
//		return ""
//	}
//	createdTime, _ := dateparse.ParseAny(dateStr)
//	createdLocal := createdTime.Local()
//
//	// Format weekday, time if less than a week old
//	age := time.Now().Sub(createdTime)
//	if age < time.Hour*24*7 {
//		// less than a week
//		formattedTime = createdLocal.Format(WeekTimeFormat)
//	} else {
//		// less than a year
//		formattedTime = createdLocal.Format(YearTimeFormat)
//	}
//	return formattedTime
//}

// FormatAge converts the given time to the current short age format h m s ago
//
// If time is less than an hour:  minutes seconds ago
// If time is less than a day:  hours minutes ago
// If time is less than a month:  days hours minutes ago
// If time is more than a month:  days hours ago
func FormatAge(dateStr string) (age string) {
	if dateStr == "" {
		return "n/a"
	}
	parsedTime, _ := dateparse.ParseAny(dateStr)
	localTime := parsedTime.Local()

	// dur := int(time.Now().Sub(localTime).Round(time.Second).Seconds())
	dur := int(time.Since(localTime).Round(time.Second).Seconds())
	days := dur / (24 * 3600)
	if days >= 1 {
		dur -= days * (24 * 3600)
	}
	hours := dur / 3600
	dur -= hours * 3600
	minutes := dur / 60
	sec := dur - minutes*60
	if days > 30 {
		age = fmt.Sprintf("%dd, %dh", days, hours)
	} else if days > 0 {
		age = fmt.Sprintf("%dd, %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		age = fmt.Sprintf("%dh %dm", hours, minutes)
	}
	age = fmt.Sprintf("%dm %ds", minutes, sec)
	return age + " ago"
}

// FormatDateTime format an iso date/time string into a human readable format
// value is an iso timestamp
// Format:
//
//		"" default is the year time format: YYYY-MM-DD, HH:MM:SS TZ
//		"S" is the shortest possible format depending on agent
//		"V" is the verbose format
//	 "AGE" is the age format like "5m 30s ago"
//
// format is default RFC822, or use "S" for a short format "weekday, time" if less than a week old
func FormatDateTime(dateStr string, format ...string) string {
	if dateStr == "" {
		return ""
	}
	createdTime, _ := dateparse.ParseAny(dateStr)
	createdLocal := createdTime.Local()
	formattedTime := ""

	if len(format) == 1 {
		// short format depending on age
		switch format[0] {
		case "AGE":
			formattedTime = FormatAge(dateStr)
		case "S":
			// Format weekday, time if less than a week old
			age := time.Since(createdTime)
			if age < time.Hour*24*7 {
				formattedTime = createdLocal.Format(WeekTimeFormat)
			} else {
				formattedTime = createdLocal.Format(YearTimeFormat)
			}
		case "V":
			formattedTime = createdLocal.Format(VerboseTimeFormat)
		default:
			formattedTime = createdLocal.Format(format[0])
		}
	} else {
		formattedTime = createdLocal.Format(YearTimeFormat)
	}
	return formattedTime
}

// FormatMSE returns a human-readable string into local time in millisec since epoc
//
// These are in timezone: time.Now().Zone()
// The short format is: StampMilli: Jan _2 15:04:05.000  (local time)
// The long format is:  YYYY-MM-DD HH:MM:SS TZ
func FormatMSE(mse int64, short bool) string {
	t := time.UnixMilli(mse).Local()
	if short {
		return t.Format(WeekTimeFormat)
	}
	return t.Format(YearTimeFormat)
}

// FormatNowUTCMilli returns the current time in UTC milliseconds
func FormatNowUTCMilli() string {
	return time.Now().UTC().Format(rfc3339Milli)
}

// FormatUTCMilli returns the given time in UTC with milliseconds,
// yyyy-mm-ddThh:mm:ss.000Z
func FormatUTCMilli(t time.Time) string {
	return t.UTC().Format(rfc3339Milli)
}
