package fakers

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/kream404/spoof/interfaces"
	"github.com/kream404/spoof/models"
)

type NumberFaker struct {
	datatype models.Type
	format   string  // decimal places (e.g. "2") or empty for raw float64
	length   int     // if set, produce a numeric string of this length
	min      float64 // lower bound (inclusive)
	max      float64 // upper bound (inclusive-ish)
	rng      *rand.Rand
	function string // e.g. "sin:period=86400", "random", "constant:value=42"
}

func (f *NumberFaker) Generate() (any, error) {
	if f.length != 0 {
		if f.min != 0 || f.max != 0 {
			lo := int(f.min)
			hi := int(f.max)
			if hi < lo {
				lo, hi = hi, lo
			}

			n := lo
			if hi > lo {
				span := hi - lo + 1
				var r int
				if f.rng != nil {
					r = f.rng.Intn(span)
				} else {
					r = rand.Intn(span)
				}
				n = lo + r
			}

			s := strconv.Itoa(n)
			return s, nil
		}

		return f.GenerateRandomNumberOfLength(f.length), nil
	}

	name, params := parseFunctionString(strings.TrimSpace(f.function))

	// Special case: constant:value as absolute numeric value
	if name == "constant" {
		if v, ok := params["value"]; ok && v != "" {
			if num, err := strconv.ParseFloat(v, 64); err == nil {
				return f.formatValue(num)
			}
		}
	}

	norm := sampleNormalized(name, params, f.rng)
	val := MapNormalizedToFloat(norm, params, f.min, f.max)
	return f.formatValue(val)
}

func (f *NumberFaker) formatValue(val float64) (any, error) {
	if strings.TrimSpace(f.format) == "" {
		return val, nil
	}
	decimals, err := strconv.Atoi(f.format)
	if err != nil {
		return nil, fmt.Errorf("invalid number format: %w", err)
	}
	format := fmt.Sprintf("%%.%df", decimals)
	return fmt.Sprintf(format, val), nil
}

func (f *NumberFaker) GenerateRandomNumberOfLength(length int) string {
	const charset = "123456789"
	result := make([]byte, length)
	for i := range result {
		idx := 0
		if f.rng != nil {
			idx = f.rng.Intn(len(charset))
		} else {
			idx = rand.Intn(len(charset))
		}
		result[i] = charset[idx]
	}
	return string(result)
}

func (f *NumberFaker) GetType() models.Type { return f.datatype }
func (f *NumberFaker) GetFormat() string    { return f.format }

func NewNumberFaker(format string, length int, min float64, max float64, rng *rand.Rand, function string) (*NumberFaker, error) {
	// basic validation
	if length == 0 && min == max {
		return nil, fmt.Errorf("invalid number config: min and max must differ when length not set")
	}
	if min > max {
		return nil, fmt.Errorf("invalid number config: min must be <= max")
	}
	fn := strings.TrimSpace(function)
	if fn == "" {
		fn = "random"
	}
	return &NumberFaker{
		datatype: models.Type("Number"),
		format:   format,
		length:   length,
		min:      min,
		max:      max,
		rng:      rng,
		function: fn,
	}, nil
}

func init() {
	RegisterFaker("number", func(field models.Field, rng *rand.Rand) (interfaces.Faker[any], error) {
		faker, err := NewNumberFaker(field.Format, field.Length, field.Min, field.Max, rng, field.Function)
		if err != nil {
			return nil, err
		}
		return faker, nil
	})
}
