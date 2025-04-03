package fakers

import (
	"log"
	"sync"

	"github.com/kream404/spoof/interfaces"
)

var registry = make(map[string]interface{})
var mu sync.Mutex

func RegisterFaker[T any](name string, faker interfaces.Faker[T]) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = faker
}

func GetFakerByName(name string) (interface{}, bool) {
	mu.Lock()
	defer mu.Unlock()
	faker, found := registry[name]
	if !found {
		log.Fatalf("Unsupported faker: %s", name)
	}
	return faker, found
}
