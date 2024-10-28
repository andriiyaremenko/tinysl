package tinysl

import (
	"context"
	"reflect"
	"runtime"
	"sync"
)

var mod = uint64(866285) // just a random uint with good enough spread

var ctxScopeKeyPool = sync.Pool{
	New: func() any {
		return &ctxScopeKey{}
	},
}

func getCtxScopeKey(ctx context.Context) *ctxScopeKey {
	key := ctxScopeKeyPool.Get().(*ctxScopeKey)
	key.ctx = ctx

	return key
}

func cleanCtxKey(key *ctxScopeKey) {
	key.clean()
	ctxScopeKeyPool.Put(key)
}

type ctxScopeKey struct {
	ctx context.Context
}

func (sk *ctxScopeKey) key() uint64 {
	return uint64(reflect.ValueOf(sk.ctx).Pointer())
}

func (sk *ctxScopeKey) pin() {
	if goHasMovingGC.Load() && sk.ctx.Err() == nil && sk.ctx.Done() != nil {
		// We don't want sk.ctx pointer value to change so we need to pin it.
		// Currently Go GC do not move values in memory (mostly) but there is no guarantee that GC implementation would't change.
		// Any reliable source on Go is telling us not to relay on consistency of values returned from reflect.Value.Pointer() and
		// in order to make it consistent we pinning context until context is done.
		pinner := &runtime.Pinner{}
		pinner.Pin(sk.ctx)
		context.AfterFunc(sk.ctx, pinner.Unpin)
	}
}

func (sk *ctxScopeKey) clean() {
	sk.ctx = nil
}

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
	newServiceScopes := func() any {
		services := make(map[int]*serviceScope)

		if len(services) == 0 {
			for _, key := range keys {
				services[key] = &serviceScope{}
			}
		}

		return services
	}
	serviceScopesPool := [9]*sync.Pool{}
	for i := range serviceScopesPool {
		func(i int) {
			serviceScopesPool[i] = &sync.Pool{New: newServiceScopes}
		}(i)
	}
	return &contextInstances{
		keys:              keys,
		serviceScopesPool: serviceScopesPool,
	}
}

type contextInstances struct {
	serviceScopesPool [9]*sync.Pool
	partitions        [9]sync.Map
	keys              []int
}

func (ci *contextInstances) get(ctxKey *ctxScopeKey, key int) (*serviceScope, int, func(), bool) {
	ctxKV := ctxKey.key()

	i := ctxKV % mod
	var partIndex int
	if n := i / 3; i == n*3 {
		if n := i / 9; i == n*9 {
			partIndex = 8
		} else if n := i / 6; i == n*6 {
			partIndex = 7
		} else {
			partIndex = 6
		}
	} else if n := (i + 1) / 3; i+1 == n*3 {
		if n := (i + 1) / 9; i+1 == n*9 {
			partIndex = 5
		} else if n := (i + 1) / 6; (i + 1) == n*6 {
			partIndex = 4
		} else {
			partIndex = 3
		}
	} else {
		if n := (i + 2) / 9; i+2 == n*9 {
			partIndex = 2
		} else if n := (i + 2) / 6; (i + 2) == n*6 {
			partIndex = 1
		} else {
			partIndex = 0
		}
	}

	servicesVal, ok := ci.partitions[partIndex].LoadOrStore(ctxKV, ci.serviceScopesPool[partIndex].Get())
	services := servicesVal.(map[int]*serviceScope)

	if !ok {
		ctxKey.pin()
		return services[key], partIndex, func() {
			ci.partitions[partIndex].Delete(ctxKV)
			for key := range services {
				services[key].lock()
				services[key].value = nil
				services[key].unlock()
			}
			ci.serviceScopesPool[partIndex].Put(services)
			cleanCtxKey(ctxKey)
		}, false
	} else {
		cleanCtxKey(ctxKey)
	}

	return services[key], partIndex, nil, true
}
