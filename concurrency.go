package tinysl

import (
	"sync"
	"sync/atomic"
)

type keyValue struct {
	value *any
	key   uintptr
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

func slToPointer(sl []*keyValue) *[]*keyValue {
	return &sl
}

func (ci *contextInstances) get(ctxKey uintptr, key uintptr) (*any, bool) {
	ptr, ok := ci.m.LoadOrStore(ctxKey, &atomic.Pointer[[]*keyValue]{})

	if !ok {
		return nil, false
	}

	if slPtr := ptr.(*atomic.Pointer[[]*keyValue]).Load(); slPtr != nil {
		for _, el := range *slPtr {
			if el.key == key {
				return el.value, true
			}
		}
	}

	return nil, false
}

func (ci *contextInstances) set(ctxKey uintptr, key uintptr, value *any) {
	ptr, _ := ci.m.Load(ctxKey)
	ptr.(*atomic.Pointer[[]*keyValue]).Store(func() *[]*keyValue {
		slPtr := ptr.(*atomic.Pointer[[]*keyValue]).Load()
		val := &keyValue{key: key, value: value}
		if slPtr == nil {
			sl := make([]*keyValue, 1, ci.c)
			sl[0] = val

			return &sl
		}

		return slToPointer(append(*slPtr, &keyValue{key: key, value: value}))
	}())
}

func (ci *contextInstances) delete(ctxKey uintptr) {
	ci.m.Delete(ctxKey)
}
