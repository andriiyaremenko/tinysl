package tinysl

import (
	"context"
	"fmt"
	"sync"
)

type keyValue struct {
	value any
	key   string
}

func getPerContextKey(ctx context.Context, key string) string {
	if key == "" {
		return fmt.Sprintf("%p", ctx)
	}

	return fmt.Sprintf("%p::%s", ctx, key)
}

func newInstances(l int) *instances {
	return &instances{m: make(map[string]any, l)}
}

type instances struct {
	m  map[string]any
	mu sync.RWMutex
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

func newContextInstances(c int) *contextInstances {
	return &contextInstances{
		c: c,
		m: make(map[string][]keyValue),
	}
}

type contextInstances struct {
	m  map[string][]keyValue
	c  int
	mu sync.RWMutex
}

func (ci *contextInstances) get(ctx context.Context, key string) (any, bool) {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	perCtxKey := getPerContextKey(ctx, "")
	sl, ok := ci.m[perCtxKey]

	if !ok {
		return nil, false
	}

	for _, el := range sl {
		if el.key == key {
			return el.value, true
		}
	}

	return nil, false
}

func (ci *contextInstances) set(ctx context.Context, key string, value any) {
	ci.mu.Lock()

	perCtxKey := getPerContextKey(ctx, "")
	keyValueVal := keyValue{key: key, value: value}

	if _, ok := ci.m[perCtxKey]; ok {
		ci.m[perCtxKey] = append(ci.m[perCtxKey], keyValueVal)
	} else {
		ci.m[perCtxKey] = make([]keyValue, 1, ci.c)
		ci.m[perCtxKey][0] = keyValueVal
	}

	ci.mu.Unlock()
}

func (ci *contextInstances) delete(ctx context.Context) {
	ci.mu.Lock()

	delete(ci.m, getPerContextKey(ctx, ""))

	ci.mu.Unlock()
}
