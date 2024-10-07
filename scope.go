package tinysl

import (
	"sync"
)

type serviceScope struct {
	value *any
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
		serviceScopesPool: sync.Pool{
			New: func() any {
				services := make(map[int]*serviceScope)

				if len(services) == 0 {
					for _, key := range keys {
						services[key] = &serviceScope{}
					}
				}

				return services
			},
		},
	}
}

type contextInstances struct {
	serviceScopesPool sync.Pool
	m                 sync.Map
	keys              []int
}

func (ci *contextInstances) get(ctxKey uintptr, key int) (*serviceScope, bool) {
	servicesVal, ok := ci.m.LoadOrStore(ctxKey, ci.serviceScopesPool.Get())
	services := servicesVal.(map[int]*serviceScope)

	return services[key], ok
}

func (ci *contextInstances) delete(ctxKey uintptr) {
	if servVal, loaded := ci.m.LoadAndDelete(ctxKey); loaded {
		serv := servVal.(map[int]*serviceScope)
		for key := range serv {
			serv[key] = &serviceScope{}
		}
		ci.serviceScopesPool.Put(serv)
	}
}
