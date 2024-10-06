package tinysl

import (
	"fmt"
	"sync"
)

var serviceScopesPool = sync.Pool{
	New: func() any {
		return make([]*serviceScope, 0)
	},
}

type serviceScope struct {
	value *any
	key   int
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

func newContextInstances(keys []int) *contextInstances {
	return &contextInstances{
		keys: keys,
	}
}

type contextInstances struct {
	m    sync.Map
	keys []int
}

func newContextScope(keys []int) []*serviceScope {
	services := serviceScopesPool.Get().([]*serviceScope)

	for _, key := range keys {
		services = append(services, &serviceScope{key: key})
	}

	return services
}

func (ci *contextInstances) get(ctxKey uintptr, key int) (*serviceScope, bool) {
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
	if servVal, loaded := ci.m.LoadAndDelete(ctxKey); loaded {
		serv := servVal.([]*serviceScope)
		serv = serv[:0]
		serviceScopesPool.Put(serv)
	}
}
