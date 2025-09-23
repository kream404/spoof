package fakers

import (
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
	now := time.Now()

	// parse function string
	name, params := parseFunctionString(strings.TrimSpace(f.function))

	// per-call interval override (supports "7d", "72h", "3600s", etc.)
	useInterval := f.interval
	if v, ok := params["interval"]; ok && v != "" {
		if d := ParseDurationExt(v, 0); d != 0 {
			useInterval = d
		}
	}

	// If interval zero -> return now/formatted now
	// if useInterval == 0 {
	// 	return formatTime(now, f.format), nil
	// }

	// Decide direction. If user specified dir param, use it.
	// Otherwise infer from sign of useInterval: negative -> past, positive -> future.
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
				return formatTime(value, f.format), nil
			}
		}
	}

	norm := sampleNormalized(name, params, f.rng)
	offset := MapNormalizedToDuration(norm, params, useInterval, dir)

	value := now.Add(offset)
	return formatTime(value, f.format), nil
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
