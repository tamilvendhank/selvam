package shared

import "time"

const localeTimeLayout = "1/2/2006, 3:04:05 PM"

func FormatDateLabel(value *time.Time, fallback string) string {
	if value == nil {
		return fallback
	}

	return value.In(time.Local).Format(localeTimeLayout)
}
