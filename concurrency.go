package tinysl

import (
	"context"
	"sync"
)

func newInstances() *instances {
	return &instances{m: make(map[string]interface{})}
}

type instances struct {
	mu sync.RWMutex
	m  map[string]interface{}
}

func (i *instances) get(key string) (interface{}, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	value, ok := i.m[key]
	return value, ok
}

func (i *instances) set(key string, value interface{}) {
	i.mu.Lock()

	i.m[key] = value

	i.mu.Unlock()
}

func newContextInstances() *contextInstances {
	return &contextInstances{m: make(map[context.Context]*instances)}
}

type contextInstances struct {
	mu sync.RWMutex
	m  map[context.Context]*instances
}

func (ci *contextInstances) get(ctx context.Context, key string) (interface{}, bool) {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	ins, ok := ci.m[ctx]
	if !ok {
		return nil, false
	}

	return ins.get(key)
}

func (ci *contextInstances) set(ctx context.Context, key string, value interface{}) {
	ci.mu.Lock()
	ins, ok := ci.m[ctx]
	if !ok {
		ins = newInstances()

		go func() {
			<-ctx.Done()

			ci.mu.Lock()
			delete(ci.m, ctx)
			ci.mu.Unlock()
		}()
	}

	ins.set(key, value)

	ci.m[ctx] = ins

	ci.mu.Unlock()
}
