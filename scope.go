package tinysl

import (
	"context"
	"reflect"
	"runtime"
	"sync"
)

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

func (sk *ctxScopeKey) key() uintptr {
	return reflect.ValueOf(sk.ctx).Pointer()
}

func (sk *ctxScopeKey) pin() {
	if sk.ctx.Err() == nil && sk.ctx.Done() != nil {
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
	m                 sync.Map
	keys              []int
}

func (ci *contextInstances) get(ctxKey *ctxScopeKey, key int) (*serviceScope, func(), bool) {
	servicesVal, ok := ci.m.LoadOrStore(ctxKey.key(), ci.serviceScopesPool.Get())
	services := servicesVal.(map[int]*serviceScope)

	if !ok {
		ctxKey.pin()
		return services[key], func() {
			ci.m.Delete(ctxKey.key())
			for key := range services {
				services[key] = &serviceScope{}
			}
			ci.serviceScopesPool.Put(services)
			cleanCtxKey(ctxKey)
		}, false
	} else {
		cleanCtxKey(ctxKey)
	}

	return services[key], nil, true
}
