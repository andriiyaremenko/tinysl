package tinysl

import (
	"fmt"
	"sync"
)

type serviceScope struct {
	value *any
	key   uintptr
	mu    sync.Mutex
}

func (cs *serviceScope) empty() bool {
	return cs.value == nil
}

func (cs *serviceScope) lock() {
	cs.mu.Lock()
}

func (cs *serviceScope) unlock() {
	cs.mu.Unlock()
}

func newContextInstances(keys []uintptr) *contextInstances {
	return &contextInstances{
		keys: keys,
	}
}

type contextInstances struct {
	m    sync.Map
	keys []uintptr
}

func newContextScope(keys []uintptr) []*serviceScope {
	services := make([]*serviceScope, len(keys))

	for i, key := range keys {
		services[i] = &serviceScope{key: key}
	}

	return services
}

func (ci *contextInstances) get(ctxKey uintptr, key uintptr) (*serviceScope, bool) {
	servicesVal, ok := ci.m.LoadOrStore(ctxKey, newContextScope(ci.keys))
	services := servicesVal.([]*serviceScope)

	for _, el := range services {
		if el.key == key {
			return el, ok
		}
	}

	panic(fmt.Sprintf("service key %d is not found", key))
}

func (ci *contextInstances) delete(ctxKey uintptr) {
	ci.m.Delete(ctxKey)
}
