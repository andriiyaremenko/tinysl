package tinysl

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

func getPerContextKey(ctx context.Context, key string) string {
	return fmt.Sprintf("%p::%s", ctx, key)
}

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
		m: make(map[string]any),
	}
}

type contextInstances struct {
	mu sync.RWMutex

	m map[string]any
}

func (ci *contextInstances) get(ctx context.Context, key string) (any, bool) {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	key = getPerContextKey(ctx, key)
	value, ok := ci.m[key]

	return value, ok
}

func (ci *contextInstances) set(ctx context.Context, key string, value any) {
	ci.mu.Lock()

	key = getPerContextKey(ctx, key)

	ci.m[key] = value

	ci.mu.Unlock()
}

func (ci *contextInstances) delete(ctx context.Context) {
	ci.mu.Lock()

	prefix := fmt.Sprintf("%p", ctx)

	for key := range ci.m {
		if strings.HasPrefix(key, prefix) {
			delete(ci.m, key)
		}
	}

	ci.mu.Unlock()
}
