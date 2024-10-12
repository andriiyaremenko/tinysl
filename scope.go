package tinysl

import (
	"context"
	"math/rand/v2"
	"reflect"
	"runtime"
	"sync"
)

var divider = rand.Uint64N(100_000_000)

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
	partitions        [9]sync.Map
	keys              []int
}

func (ci *contextInstances) get(ctxKey *ctxScopeKey, key int) (*serviceScope, int, func(), bool) {
	ctxKV := ctxKey.key()

	i := ctxKey.key() / divider
	var partIndex int
	if n := i / 3; i == n*3 {
		if n := i / 9; i == n*9 {
			partIndex = 8
		} else if n := i / 6; i == n*6 {
			partIndex = 7
		} else {
			partIndex = 6
		}
	} else if n := i / 2; i == n*2 {
		if n := i / 7; i == n*7 {
			partIndex = 5
		} else if n := i / 5; i == n*5 {
			partIndex = 5
		} else if n := i / 4; i == n*4 {
			partIndex = 4
		} else {
			partIndex = 3
		}
	} else {
		if n := i / 7; i == n*7 {
			partIndex = 2
		} else if n := i / 5; i == n*5 {
			partIndex = 2
		} else if n := (i - 1) / 3; i-1 == n*3 {
			partIndex = 1
		} else {
			partIndex = 0
		}
	}

	servicesVal, ok := ci.partitions[partIndex].LoadOrStore(ctxKV, ci.serviceScopesPool.Get())
	services := servicesVal.(map[int]*serviceScope)

	if !ok {
		ctxKey.pin()
		return services[key], partIndex, func() {
			ci.partitions[partIndex].Delete(ctxKV)
			for key := range services {
				services[key] = &serviceScope{}
			}
			ci.serviceScopesPool.Put(services)
			cleanCtxKey(ctxKey)
		}, false
	} else {
		cleanCtxKey(ctxKey)
	}

	return services[key], partIndex, nil, true
}
