package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// LoadLocation parses a timezone string into a time.Location.
//
// Supported inputs:
// - "" or "UTC" => time.UTC
// - IANA name (e.g. "Asia/Jakarta")
// - Fixed offsets: "UTC+7", "UTC+07", "UTC+07:00", "UTC-3", ...
//
// This is presentation-only. Core timestamps are still stored as absolute instants.
func LoadLocation(tz string) (*time.Location, error) {
	tz = strings.TrimSpace(tz)
	if tz == "" || strings.EqualFold(tz, "UTC") {
		return time.UTC, nil
	}

	// Fixed offset: UTC+7, UTC+07, UTC+07:00, UTC-03:30
	up := strings.ToUpper(tz)
	if strings.HasPrefix(up, "UTC+") || strings.HasPrefix(up, "UTC-") {
		sign := 1
		rest := strings.TrimPrefix(up, "UTC")
		if strings.HasPrefix(rest, "+") {
			rest = strings.TrimPrefix(rest, "+")
			sign = 1
		} else if strings.HasPrefix(rest, "-") {
			rest = strings.TrimPrefix(rest, "-")
			sign = -1
		}
		rest = strings.TrimSpace(rest)

		hours := 0
		mins := 0
		if strings.Contains(rest, ":") {
			parts := strings.SplitN(rest, ":", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid timezone offset %q", tz)
			}
			h, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid timezone offset %q", tz)
			}
			m, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid timezone offset %q", tz)
			}
			hours, mins = h, m
		} else {
			h, err := strconv.Atoi(rest)
			if err != nil {
				return nil, fmt.Errorf("invalid timezone offset %q", tz)
			}
			hours = h
			mins = 0
		}
		if hours < 0 {
			hours = -hours
		}
		if mins < 0 {
			mins = -mins
		}
		if hours > 14 || mins >= 60 {
			return nil, fmt.Errorf("invalid timezone offset %q", tz)
		}
		offset := sign * ((hours * 60 * 60) + (mins * 60))
		name := fmt.Sprintf("UTC%+02d:%02d", sign*hours, mins)
		return time.FixedZone(name, offset), nil
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %q: %w", tz, err)
	}
	return loc, nil
}
