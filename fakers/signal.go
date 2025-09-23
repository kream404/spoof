package fakers

import (
	"math"
	"math/rand"
	"strings"
	"time"
)

func parseFunctionString(s string) (name string, params map[string]string) {
	name = "constant"
	params = map[string]string{}
	if s == "" {
		return
	}
	parts := strings.SplitN(s, ":", 2)
	name = strings.ToLower(strings.TrimSpace(parts[0]))
	if len(parts) == 1 {
		return
	}
	for _, kv := range strings.Split(parts[1], ",") {
		kv = strings.TrimSpace(kv)
		if kv == "" {
			continue
		}
		p := strings.SplitN(kv, "=", 2)
		if len(p) != 2 {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(p[0]))
		v := strings.TrimSpace(p[1])
		params[k] = v
	}
	return
}

// sampleNormalized returns a value in [0,1] for the provided function name and params.
// rng may be nil: fallback to math/rand.
//
// Period param accepts either a plain numeric value in seconds (e.g. "60")
// OR a duration string supported by ParseDurationExt (e.g. "7d", "72h", "1.5d").
//
// Jitter params:
//   - "jitter" (probability 0..1) enables occasional outliers
//   - "jitter_type" in {"scale","edge","spike"} controls how outliers are created
//   - "jitter_amp" multiplier used for "scale" type (default 3.0)
func sampleNormalized(fn string, params map[string]string, rng *rand.Rand) float64 {
	var base float64

	switch fn {
	case "random":
		if rng != nil {
			base = rng.Float64()
		} else {
			base = rand.Float64()
		}

	case "sin":
		// accepts either numeric seconds or human-friendly duration strings like "7d", "72h"
		period := 60.0
		if pstr := params["period"]; pstr != "" {
			if d := ParseDurationExt(pstr, 0); d > 0 {
				period = d.Seconds()
			} else {
				period = parseFloat(pstr, 60.0)
			}
		} else {
			period = parseFloat(params["period"], 60.0)
		}
		if period <= 0 {
			period = 60.0
		}

		phaseDeg := parseFloat(params["phase"], 0.0)
		t := float64(time.Now().UnixNano()) / 1e9
		phase := 2*math.Pi*(t/period) + (phaseDeg * math.Pi / 180.0)
		s := math.Sin(phase)   // [-1,1]
		base = (s + 1.0) / 2.0 // [0,1]

	case "linear":
		period := 60.0
		if pstr := params["period"]; pstr != "" {
			if d := ParseDurationExt(pstr, 0); d > 0 {
				period = d.Seconds()
			} else {
				period = parseFloat(pstr, 60.0)
			}
		} else {
			period = parseFloat(params["period"], 60.0)
		}
		if period <= 0 {
			period = 60.0
		}
		t := float64(time.Now().UnixNano()) / 1e9
		base = math.Mod(t, period) / period // [0,1)

	case "constant":
		if v, ok := params["valuenorm"]; ok && v != "" {
			n := parseFloat(v, 0.0)
			if n < 0 {
				n = 0
			} else if n > 1 {
				n = 1
			}
			base = n
		} else {
			// constant returns normalized 0 by default (caller maps it)
			base = 0.0
		}

	default:
		base = 0.0
	}

	// apply jitter
	jitterProb := parseFloat(params["jitter"], 0.0)
	if jitterProb > 0 {
		var r float64
		if rng != nil {
			r = rng.Float64()
		} else {
			r = rand.Float64()
		}
		if r < jitterProb {
			// produce an outlier
			jt := strings.ToLower(strings.TrimSpace(params["jitter_type"]))
			if jt == "" {
				jt = "scale"
			}
			jam := parseFloat(params["jitter_amp"], 3.0) // scale multiplier for "scale" type

			switch jt {
			case "edge":
				// hard edge: return either 0 or 1 (equal chance)
				var r2 float64
				if rng != nil {
					r2 = rng.Float64()
				} else {
					r2 = rand.Float64()
				}
				if r2 < 0.5 {
					return 0.0
				}
				return 1.0

			case "spike":
				// small spikes near edges: choose side then pick a small range
				var r2, r3 float64
				if rng != nil {
					r2 = rng.Float64()
					r3 = rng.Float64()
				} else {
					r2 = rand.Float64()
					r3 = rand.Float64()
				}
				if r2 < 0.5 {
					// near 0: [0, 0.1]
					return math.Min(0.1*r3, 1.0)
				}
				// near 1: [0.9, 1.0]
				return 1.0 - math.Min(0.1*r3, 1.0)

			default: // "scale"
				out := base * jam

				if strings.ToLower(strings.TrimSpace(params["clamp"])) == "false" {
					return out
				}

				if out > 1.0 {
					out = 1.0
				} else if out < 0.0 {
					out = 0.0
				}
				return out
			}
		}
	}

	return base
}
