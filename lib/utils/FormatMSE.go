package utils

import "time"

var ShortFormat = time.StampMilli
var LongFormat = "2006-01-02 15:04:05.000 -0700"

// RFC3339Milli is the M2M format for ISO8601/RFC3339 timestamps in millisecond
const RFC3339Milli = "2006-01-02T15:04:05.999-0700"

// FormatMSE returns a human readable string of the time in millisec since epoc
// The short format is: StampMilli: Jan _2 15:04:05.000  (local time)
// The long format is:  RFC1123Z: Mon, 02 Jan 2006 15:04:05 -07:00
func FormatMSE(mse int64, short bool) string {
	t := time.UnixMilli(mse)
	if short {
		return t.Format(ShortFormat)
	}
	return t.Format(LongFormat)
}
