package fakers

import (
	"strconv"
	"strings"
	"time"
)

func ParseDurationExt(s string, def time.Duration) time.Duration {
	if s == "" {
		return def
	}
	s = strings.TrimSpace(s)
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}

	if len(s) == 0 {
		return def
	}
	last := s[len(s)-1]
	num := strings.TrimSpace(s[:len(s)-1])

	switch last {
	case 'd', 'D':
		if num == "" {
			return def
		}
		if f, err := strconv.ParseFloat(num, 64); err == nil {
			return time.Duration(f * 24 * float64(time.Hour))
		}
	case 'w', 'W':
		if num == "" {
			return def
		}
		if f, err := strconv.ParseFloat(num, 64); err == nil {
			return time.Duration(f * 7 * 24 * float64(time.Hour))
		}
	default:
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return time.Duration(f * float64(time.Second))
		}
	}

	return def
}
