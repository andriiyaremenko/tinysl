package tinysl

import (
	"context"
	"fmt"
	"slices"
	"sync"
)

type keyValue[T any] struct {
	value T
	key   string
}

func getPerContextKey(ctx context.Context, key string) string {
	if key == "" {
		return fmt.Sprintf("%p", ctx)
	}

	return fmt.Sprintf("%p::%s", ctx, key)
}

func newInstances() *instances {
	return &instances{m: make(map[string]any)}
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

func newContextInstances() *contextInstances {
	return &contextInstances{
		m: make([]keyValue[[]keyValue[any]], 1),
	}
}

type contextInstances struct {
	m  []keyValue[[]keyValue[any]]
	mu sync.RWMutex
}

func (ci *contextInstances) get(ctx context.Context, key string) (value any, ok bool) {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	ctxKey := getPerContextKey(ctx, "")
	i, ok := slices.BinarySearchFunc(ci.m, ctxKey, perContextSortingOrder)

	if !ok || ci.m[i].key != ctxKey {
		return nil, false
	}

	ok = false
	for _, vKeyVal := range ci.m[i].value {
		if vKeyVal.key == key {
			value = vKeyVal.value
			ok = true
		}
	}

	return value, ok
}

func (ci *contextInstances) set(ctx context.Context, key string, value any) {
	ci.mu.Lock()

	ctxKey := getPerContextKey(ctx, "")
	vKeyVal := keyValue[any]{value, key}

	i, ok := slices.BinarySearchFunc(ci.m, ctxKey, perContextSortingOrder)

	if ok && ci.m[i].key == ctxKey {
		ci.m[i].value = append(ci.m[i].value, vKeyVal)
	} else {
		ci.m = slices.Insert(ci.m, i, keyValue[[]keyValue[any]]{key: ctxKey, value: []keyValue[any]{vKeyVal}})
	}

	ci.mu.Unlock()
}

func (ci *contextInstances) delete(ctx context.Context) {
	ci.mu.Lock()

	ctxKey := getPerContextKey(ctx, "")

	i, ok := slices.BinarySearchFunc(ci.m, ctxKey, perContextSortingOrder)

	if ok && ci.m[i].key == ctxKey {
		ci.m = slices.Delete(ci.m, i, i+1)
	}

	ci.mu.Unlock()
}

func perContextSortingOrder(el keyValue[[]keyValue[any]], target string) int {
	switch {
	case el.key < target:
		return -1
	case el.key > target:
		return 1
	default:
		return 0
	}
}
