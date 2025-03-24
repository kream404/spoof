package fakers

import (
	"github.com/kream404/scratch/interfaces"
	"sync"
)

var registry = make(map[string]interface{}) // This will hold interface{} to allow different generics
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
	return faker, found
}
