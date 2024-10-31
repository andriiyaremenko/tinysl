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

type contextScope struct {
	cleanup  *cleanupNode
	services []*serviceScope
}

func newContextInstances(size int32, buildCleanupNode func() *cleanupNode) *contextInstances {
	return &contextInstances{
		serviceScopesPool: sync.Pool{
			New: func() any {
				services := make([]*serviceScope, size)

				for i := range services {
					services[i] = &serviceScope{}
				}

				return &contextScope{services: services, cleanup: buildCleanupNode()}
			},
		},
	}
}

type contextInstances struct {
	serviceScopesPool sync.Pool
	partitions        [18]sync.Map
}

func (ci *contextInstances) get(ctx context.Context, key int32) (*serviceScope, *cleanupNode) {
	ctxKey := getCtxScopeKey(ctx)
	ctxKV := ctxKey.key()

	i := ctxKV % mod
	var partIndex int
	if n := i / 3; i == n*3 {
		if n := i / 9; i == n*9 {
			if n := i / 18; i == n*18 {
				partIndex = 17
			} else {
				partIndex = 16
			}
		} else if n := i / 6; i == n*6 {
			if n := i / 12; i == n*12 {
				partIndex = 15
			} else {
				partIndex = 14
			}
		} else {
			if n := (i + 1) / 4; i+1 == n*4 {
				partIndex = 13
			} else {
				partIndex = 12
			}
		}
	} else if n := (i + 1) / 3; i+1 == n*3 {
		if n := (i + 1) / 9; i+1 == n*9 {
			if n := (i + 1) / 18; i+1 == n*18 {
				partIndex = 11
			} else {
				partIndex = 10
			}
		} else if n := (i + 1) / 6; (i + 1) == n*6 {
			if n := (i + 1) / 12; i+1 == n*12 {
				partIndex = 9
			} else {
				partIndex = 8
			}
		} else {
			if n := i / 4; i == n*4 {
				partIndex = 7
			} else {
				partIndex = 6
			}
		}
	} else {
		if n := (i + 2) / 9; i+2 == n*9 {
			if n := (i + 2) / 18; i+2 == n*18 {
				partIndex = 5
			} else {
				partIndex = 4
			}
		} else if n := (i + 2) / 6; (i + 2) == n*6 {
			if n := (i + 2) / 12; i+2 == n*12 {
				partIndex = 3
			} else {
				partIndex = 2
			}
		} else {
			if n := (i + 3) / 4; i+3 == n*4 {
				partIndex = 1
			} else {
				partIndex = 0
			}
		}
	}

	scopeVal, ok := ci.partitions[partIndex].LoadOrStore(ctxKV, ci.serviceScopesPool.Get())
	scope := scopeVal.(*contextScope)

	if !ok {
		ctxKey.pin()
		context.AfterFunc(ctx, func() {
			if scopeVal, ok := ci.partitions[partIndex].LoadAndDelete(ctxKV); ok {
				scope := scopeVal.(*contextScope)

				if !scope.cleanup.empty() {
					Cleanup(scope.cleanup.clean).CallWithRecovery(PerContext)
				}

				for key := range scope.services {
					scope.services[key].lock()
					scope.services[key].value = nil
					scope.services[key].unlock()
				}

				ci.serviceScopesPool.Put(scope)
				cleanCtxKey(ctxKey)
			}
		})
	} else {
		cleanCtxKey(ctxKey)
	}

	return scope.services[key], scope.cleanup
}
