package fakers

import (
	"strconv"
	"strings"
	"time"

	"math/rand"

	"github.com/kream404/spoof/interfaces"
	"github.com/kream404/spoof/models"
)

type TimestampFaker struct {
	datatype models.Type
	format   string
	interval time.Duration // default magnitude for offsets (can be negative to imply past)
	rng      *rand.Rand
	function string // e.g. "sin:period=7d,dir=both,amplitude=2,center=-1d"
}

func (f *TimestampFaker) Generate() (any, error) {
	now := time.Now().UTC().Truncate(time.Second)
	name, params := parseFunctionString(strings.TrimSpace(f.function))

	useInterval := f.interval
	if v, ok := params["interval"]; ok && v != "" {
		if d := ParseDurationExt(v, 0); d != 0 {
			useInterval = d
		}
	}

	dir := strings.ToLower(strings.TrimSpace(params["dir"]))
	if dir == "" {
		if useInterval < 0 {
			dir = "past"
		} else {
			dir = "future"
		}
	}

	if name == "constant" {
		if v := params["value"]; v != "" {
			// allow negative leading sign too
			sign := 1.0
			clean := v
			if strings.HasPrefix(clean, "-") {
				sign = -1.0
				clean = strings.TrimPrefix(clean, "-")
			}
			if d := ParseDurationExt(clean, 0); d != 0 {
				var offset time.Duration
				switch dir {
				case "past":
					offset = -time.Duration(sign * float64(d))
				case "both":
					offset = 0
				default:
					offset = time.Duration(sign * float64(d))
				}
				value := now.Add(offset)
				value = applyDateTimeOverrides(value, params, dir)
				return formatTime(value, f.format), nil
			}
		}
	}

	norm := sampleNormalized(name, params, f.rng)
	offset := MapNormalizedToDuration(norm, params, useInterval, dir)

	value := now.Add(offset)
	value = applyDateTimeOverrides(value, params, dir)
	return formatTime(value, f.format), nil
}

func applyDateTimeOverrides(t time.Time, params map[string]string, dir string) time.Time {
	tt := t.UTC()

	if ds := strings.TrimSpace(params["date"]); ds != "" {
		// Expect YYYY-MM-DD
		if d, err := time.ParseInLocation("2006-01-02", ds, time.UTC); err == nil {
			tt = time.Date(d.Year(), d.Month(), d.Day(), tt.Hour(), tt.Minute(), tt.Second(), 0, time.UTC)
		}
	}

	if ws := strings.TrimSpace(params["weekday"]); ws != "" {
		if wd, ok := parseWeekday(ws); ok {
			tt = shiftToWeekday(tt, wd, dir)
		}
	}

	if ts := strings.TrimSpace(params["time"]); ts != "" {
		if h, m, s, ok := parseClock(ts); ok {
			tt = time.Date(tt.Year(), tt.Month(), tt.Day(), h, m, s, 0, time.UTC)
		}
	} else {
		h, hasH := parseIntParam(params, "hour")
		m, hasM := parseIntParam(params, "minute")
		s, hasS := parseIntParam(params, "second")

		if hasH || hasM || hasS {
			nh, nm, ns := tt.Hour(), tt.Minute(), tt.Second()
			if hasH {
				nh = clamp(h, 0, 23)
			}
			if hasM {
				nm = clamp(m, 0, 59)
			}
			if hasS {
				ns = clamp(s, 0, 59)
			}
			tt = time.Date(tt.Year(), tt.Month(), tt.Day(), nh, nm, ns, 0, time.UTC)
		}
	}

	return tt
}

func parseWeekday(s string) (time.Weekday, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "mon", "monday":
		return time.Monday, true
	case "tue", "tues", "tuesday":
		return time.Tuesday, true
	case "wed", "wednesday":
		return time.Wednesday, true
	case "thu", "thur", "thurs", "thursday":
		return time.Thursday, true
	case "fri", "friday":
		return time.Friday, true
	case "sat", "saturday":
		return time.Saturday, true
	case "sun", "sunday":
		return time.Sunday, true
	default:
		return time.Sunday, false
	}
}

func shiftToWeekday(t time.Time, target time.Weekday, dir string) time.Time {
	cur := t.Weekday()
	if cur == target {
		return t
	}

	// distance forward in [1..6]
	forward := (int(target) - int(cur) + 7) % 7
	if forward == 0 {
		forward = 7
	}
	// distance backward in [-6..-1]
	backward := forward - 7

	switch strings.ToLower(strings.TrimSpace(dir)) {
	case "past":
		return t.AddDate(0, 0, backward)
	case "future":
		return t.AddDate(0, 0, forward)
	case "both":
		// choose nearest; tie -> past
		if forward < -backward {
			return t.AddDate(0, 0, forward)
		}
		return t.AddDate(0, 0, backward)
	default:
		// default behave like future
		return t.AddDate(0, 0, forward)
	}
}

func parseClock(s string) (h, m, sec int, ok bool) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, 0, 0, false
	}

	hh, err1 := strconv.Atoi(parts[0])
	mm, err2 := strconv.Atoi(parts[1])
	ss := 0
	var err3 error
	if len(parts) == 3 {
		ss, err3 = strconv.Atoi(parts[2])
	}
	if err1 != nil || err2 != nil || (len(parts) == 3 && err3 != nil) {
		return 0, 0, 0, false
	}

	return clamp(hh, 0, 23), clamp(mm, 0, 59), clamp(ss, 0, 59), true
}

func parseIntParam(params map[string]string, key string) (int, bool) {
	v, ok := params[key]
	if !ok {
		return 0, false
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return i, true
}

func clamp(x, lo, hi int) int {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func formatTime(t time.Time, format string) any {
	if format != "" {
		return t.Format(format)
	}
	return t
}

func (f *TimestampFaker) GetType() models.Type { return f.datatype }
func (f *TimestampFaker) GetFormat() string    { return f.format }

func NewTimestampFaker(format string, intervalSeconds int64, rng *rand.Rand, function string) *TimestampFaker {
	fn := strings.TrimSpace(function)
	if fn == "" {
		fn = "constant"
	}
	return &TimestampFaker{
		datatype: models.Type("Timestamp"),
		format:   format,
		interval: time.Duration(intervalSeconds) * time.Second,
		rng:      rng,
		function: fn,
	}
}

func init() {
	RegisterFaker("timestamp", func(field models.Field, rng *rand.Rand) (interfaces.Faker[any], error) {
		return NewTimestampFaker(field.Format, field.Interval, rng, field.Function), nil
	})
}
