package tinysl

import (
	"sync"
)

type keyValue struct {
	value *any
	key   uintptr
}

type keyValueSyncSlice struct {
	slPtr []*keyValue
	rwMu  sync.RWMutex
}

func getPerContextKey(ctxKey, key uintptr) [2]uintptr {
	return [2]uintptr{ctxKey, key}
}

func newContextInstances(c int) *contextInstances {
	return &contextInstances{
		c: c,
	}
}

type contextInstances struct {
	m sync.Map
	c int
}

func (ci *contextInstances) get(ctxKey uintptr, key uintptr) (*any, bool) {
	syncSl, ok := ci.m.LoadOrStore(ctxKey, &keyValueSyncSlice{slPtr: make([]*keyValue, 0, ci.c)})

	if !ok {
		return nil, false
	}

	syncSl.(*keyValueSyncSlice).rwMu.RLock()
	defer syncSl.(*keyValueSyncSlice).rwMu.RUnlock()

	for _, el := range syncSl.(*keyValueSyncSlice).slPtr {
		if el.key == key {
			return el.value, true
		}
	}

	return nil, false
}

func (ci *contextInstances) set(ctxKey uintptr, key uintptr, value *any) {
	syncSl, _ := ci.m.Load(ctxKey)

	syncSl.(*keyValueSyncSlice).rwMu.Lock()
	syncSl.(*keyValueSyncSlice).slPtr = append(syncSl.(*keyValueSyncSlice).slPtr, &keyValue{value, key})
	syncSl.(*keyValueSyncSlice).rwMu.Unlock()
}

func (ci *contextInstances) delete(ctxKey uintptr) {
	ci.m.Delete(ctxKey)
}
