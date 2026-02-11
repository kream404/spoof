package fakers

import (
	"math"
	"strconv"
	"strings"
	"time"
)

func parseFloat(s string, def float64) float64 {
	if s == "" {
		return def
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return f
}

func parseAmplitude(params map[string]string) float64 {
	a := parseFloat(params["amplitude"], 1.0)
	if a < 0 {
		return math.Abs(a)
	}
	return a
}

// --------- Numeric mapping (for NumberFaker etc.) ---------
// MapNormalizedToFloat maps a normalized sample (0..1) into [min,max], applying:
//   - amplitude multiplier (param "amplitude", default 1.0) applied to half-range
//   - center shift (param "center", parsed as float, default 0.0)
//   - optional clamping controlled by param "clamp" (default "true", set "false" to disable)
//
// The returned value is not rounded/formatted by this helper (formating is up to the caller).
func MapNormalizedToFloat(norm float64, params map[string]string, min, max float64) float64 {
	if min > max {
		min, max = max, min
	}
	amp := parseAmplitude(params)
	center := parseFloat(params["center"], 0.0)

	// midpoint and half-range
	mid := (min + max) / 2.0
	half := (max - min) / 2.0 * amp

	// convert norm [0,1] to [-1,1]
	x11 := 2*norm - 1
	val := mid + center + x11*half

	// clamp unless user explicitly disables
	if strings.ToLower(params["clamp"]) != "false" {
		if val < min {
			val = min
		} else if val > max {
			val = max
		}
	}
	return val
}

// --------- Duration mapping (for TimestampFaker etc.) ---------
// MapNormalizedToDuration maps a normalized sample (0..1) into a time.Duration offset.
// Params:
//   - params["amplitude"] (multiplier, default 1.0)
//   - params["center"] (duration string, e.g. "1d" or "-2h", optional)
//   - dir: "past" | "future" | "both" (affects sign mapping)
func MapNormalizedToDuration(norm float64, params map[string]string, base time.Duration, dir string) time.Duration {

	amp := parseAmplitude(params)

	// infer direction if not provided
	d := strings.ToLower(strings.TrimSpace(dir))
	if d == "" {
		if base < 0 {
			d = "past"
		} else {
			d = "future"
		}
	}

	// magnitude controls range size
	mag := time.Duration(math.Abs(float64(base)))
	effective := time.Duration(float64(mag) * amp)

	// parse center
	center := time.Duration(0)
	if cs := strings.TrimSpace(params["center"]); cs != "" {
		if cd := ParseDurationExt(cs, 0); cd != 0 {
			center = cd
		}
	}

	var result time.Duration

	switch d {
	case "past":
		// [-effective, 0]
		result = time.Duration((norm-1.0)*float64(effective)) + center

	case "both":
		// [-effective, +effective]
		result = time.Duration((2*norm-1.0)*float64(effective)) + center

	default: // "future"
		// [0, +effective]
		result = time.Duration(norm*float64(effective)) + center
	}

	return result
}
