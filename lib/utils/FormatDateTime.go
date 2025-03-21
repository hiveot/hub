package utils

import (
	"github.com/araddon/dateparse"
	"time"
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

// FormatDateTime format an iso date/time string into a human readable format
// value is an iso timestamp
// Format:
//
//	"" default is the year time format: YYYY-MM-DD, HH:MM:SS TZ
//	"S" is the shortest possible format depending on agent
//	"V" is the verbose format
//
// format is default RFC822, or use "S" for a short format "weekday, time" if less than a week old
func FormatDateTime(dateStr string, format ...string) string {
	if dateStr == "" {
		return ""
	}
	createdTime, _ := dateparse.ParseAny(dateStr)
	createdLocal := createdTime.Local()
	formattedTime := ""

	if format != nil && len(format) == 1 {
		// short format depending on age
		if format[0] == "S" {
			// Format weekday, time if less than a week old
			age := time.Now().Sub(createdTime)
			if age < time.Hour*24*7 {
				formattedTime = createdLocal.Format(WeekTimeFormat)
			} else {
				formattedTime = createdLocal.Format(YearTimeFormat)
			}
		} else if format[0] == "V" {
			formattedTime = createdLocal.Format(VerboseTimeFormat)
		} else {
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
