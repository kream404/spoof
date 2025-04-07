package fakers

import (
	"log"
	"math/rand"
	"sync"

	"github.com/kream404/spoof/interfaces"
	"github.com/kream404/spoof/models"
)

type FakerFactory func(field models.Field, rng *rand.Rand) (interfaces.Faker[any], error)

var registry = make(map[string]FakerFactory)
var mu sync.Mutex

func RegisterFaker(name string, factory FakerFactory) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = factory
}

func GetFakerByName(name string) (FakerFactory, bool) {
	mu.Lock()
	defer mu.Unlock()
	factory, found := registry[name]
	if !found {
		log.Fatalf("Unsupported faker: %s", name)
	}
	return factory, found
}
