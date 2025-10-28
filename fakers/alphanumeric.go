package fakers

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/kream404/spoof/interfaces"
	"github.com/kream404/spoof/models"

	"github.com/lucasjones/reggen"
)

type AlphanumericFaker struct {
	datatype models.Type
	format   string // "upper" | "lower" | "mixed" (default)
	rng      *rand.Rand
	length   int
	regex    string
}

func (f *AlphanumericFaker) Generate() (any, error) {
	if f.regex != "" {
		str, _ := reggen.Generate(f.regex, 10)
		return str, nil
	}

	if f.length <= 0 {
		return "", fmt.Errorf("alphanumeric: invalid length %d; must be > 0", f.length)
	}

	var charset string
	switch strings.ToLower(strings.TrimSpace(f.format)) {
	case "upper":
		charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	case "lower":
		charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	case "", "mixed":
		fallthrough
	default:
		charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	}

	var b = make([]byte, f.length)
	for i := 0; i < f.length; i++ {
		b[i] = charset[f.rng.Intn(len(charset))]
	}

	return string(b), nil
}

func (f *AlphanumericFaker) GetType() models.Type {
	return f.datatype
}

func (f *AlphanumericFaker) GetFormat() string {
	return f.format
}

func NewAlphanumericFaker(format string, length int, regex string, rng *rand.Rand) (*AlphanumericFaker, error) {
	if length <= 0 && regex == "" {
		return nil, fmt.Errorf("alphanumeric: length must be > 0 (got %d)", length)
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	mode := strings.ToLower(strings.TrimSpace(format))
	switch mode {
	case "", "mixed", "upper", "lower":
		// ok
	default:
		return nil, fmt.Errorf("alphanumeric: invalid format %q (expected \"upper\", \"lower\", or \"mixed\")", format)
	}

	return &AlphanumericFaker{
		datatype: models.Type("alphanumeric"),
		format:   mode, // keep normalized
		rng:      rng,
		length:   length,
		regex:    regex,
	}, nil
}

func init() {
	RegisterFaker("alphanumeric", func(field models.Field, rng *rand.Rand) (interfaces.Faker[any], error) {
		faker, err := NewAlphanumericFaker(field.Format, field.Length, field.Regex, rng)
		if err != nil {
			return nil, err
		}
		return faker, nil
	})
}
