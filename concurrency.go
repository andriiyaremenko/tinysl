package tinysl

import (
	"context"
	"sync"
)

func newInstances() *instances {
	return &instances{m: make(map[string]any)}
}

type instances struct {
	mu sync.RWMutex
	m  map[string]any
}

func (i *instances) get(key string) (any, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	value, ok := i.m[key]

	return value, ok
}

func (i *instances) set(key string, value any) {
	i.mu.Lock()

	i.m[key] = value

	i.mu.Unlock()
}

func newContextInstances() *contextInstances {
	return &contextInstances{
		m: make(map[context.Context]map[string]any),
	}
}

type contextInstances struct {
	mu sync.RWMutex

	m map[context.Context]map[string]any
}

func (ci *contextInstances) get(ctx context.Context, key string) (any, bool) {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	ins, ok := ci.m[ctx]

	if !ok {
		return nil, false
	}

	value, ok := ins[key]

	return value, ok
}

func (ci *contextInstances) set(ctx context.Context, key string, value any) {
	ci.mu.Lock()

	if _, ok := ci.m[ctx]; !ok {
		ci.m[ctx] = make(map[string]any)
	}

	ci.m[ctx][key] = value

	ci.mu.Unlock()
}

func (ci *contextInstances) delete(ctx context.Context) {
	ci.mu.Lock()

	delete(ci.m, ctx)

	ci.mu.Unlock()
}
