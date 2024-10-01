package tinysl

import (
	"sync"
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
	sl, ok := ci.m.LoadOrStore(ctxKey, slToPointer(make([]*keyValue, 0, ci.c)))

	if !ok {
		return nil, false
	}

	for _, el := range *sl.(*[]*keyValue) {
		if el.key == key {
			return el.value, true
		}
	}

	return nil, false
}

func (ci *contextInstances) set(ctxKey uintptr, key uintptr, value *any) {
	sl, _ := ci.m.Load(ctxKey)
	*sl.(*[]*keyValue) = append(*sl.(*[]*keyValue), &keyValue{key: key, value: value})
}

func (ci *contextInstances) delete(ctxKey uintptr) {
	ci.m.Delete(ctxKey)
}
