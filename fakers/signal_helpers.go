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
	baseMag := time.Duration(math.Abs(float64(base)))
	amp := parseAmplitude(params)
	effective := time.Duration(float64(baseMag) * amp)

	center := time.Duration(0)
	if cs, ok := params["center"]; ok && strings.TrimSpace(cs) != "" {
		clean := strings.TrimSpace(cs)
		sign := 1.0
		if strings.HasPrefix(clean, "-") {
			sign = -1.0
			clean = strings.TrimPrefix(clean, "-")
		}
		if d := ParseDurationExt(clean, 0); d != 0 {
			center = time.Duration(sign * float64(d))
		}
	}

	// map based on dir
	switch strings.ToLower(strings.TrimSpace(dir)) {
	case "past":
		// offset = -effective + norm*effective = (norm-1)*effective
		return time.Duration((norm-1.0)*float64(effective)) + center
	case "both":
		// [0,1] -> [-effective, +effective]
		return time.Duration((2*norm-1)*float64(effective)) + center
	default:
		// norm=0 -> 0, norm=1 -> +effective
		return time.Duration(norm*float64(effective)) + center
	}
}
