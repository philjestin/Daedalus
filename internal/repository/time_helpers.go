package repository

import (
	"fmt"
	"strings"
	"time"
)

// scannable is implemented by *sql.Row and *sql.Rows.
type scannable interface {
	Scan(dest ...interface{}) error
}

// scanRow wraps Scan to automatically handle SQLite TEXT → time.Time conversion.
// Any *time.Time or **time.Time destination is wrapped with a scanner that parses
// the stored string value into a time.Time.
func scanRow(s scannable, dest ...interface{}) error {
	wrapped := make([]interface{}, len(dest))
	for i, d := range dest {
		switch v := d.(type) {
		case *time.Time:
			wrapped[i] = &sqliteTime{dest: v}
		case **time.Time:
			wrapped[i] = &sqliteNullTime{dest: v}
		default:
			wrapped[i] = d
		}
	}
	return s.Scan(wrapped...)
}

// sqliteTime scans a SQLite TEXT value into a time.Time.
type sqliteTime struct {
	dest *time.Time
}

func (s *sqliteTime) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	switch v := src.(type) {
	case time.Time:
		*s.dest = v
		return nil
	case string:
		t, err := parseTime(v)
		if err != nil {
			return err
		}
		*s.dest = t
		return nil
	default:
		return fmt.Errorf("cannot scan %T into time.Time", src)
	}
}

// sqliteNullTime scans a SQLite TEXT value into a *time.Time (nullable).
type sqliteNullTime struct {
	dest **time.Time
}

func (s *sqliteNullTime) Scan(src interface{}) error {
	if src == nil {
		*s.dest = nil
		return nil
	}
	switch v := src.(type) {
	case time.Time:
		*s.dest = &v
		return nil
	case string:
		if v == "" {
			*s.dest = nil
			return nil
		}
		t, err := parseTime(v)
		if err != nil {
			return err
		}
		*s.dest = &t
		return nil
	default:
		return fmt.Errorf("cannot scan %T into *time.Time", src)
	}
}

// parseTime attempts to parse a time string using multiple formats.
var timeFormats = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05Z07:00",
	"2006-01-02 15:04:05-07:00",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

func parseTime(v string) (time.Time, error) {
	// Strip Go's monotonic clock reading (e.g., " m=+0.123456789")
	if idx := strings.Index(v, " m="); idx > 0 {
		v = v[:idx]
	}
	// Strip timezone name if present (e.g., " MST" before the offset)
	// Format: "2026-01-26 21:57:31.12551 -0700 MST"
	// We want: "2026-01-26 21:57:31.12551 -0700"

	for _, format := range timeFormats {
		if t, err := time.Parse(format, v); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %q", v)
}
