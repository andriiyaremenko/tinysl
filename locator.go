// NOTE: building dependency graph for PerContext for every service
// and assembling them within single call to (*locator).getSingleton
// proved to be ineffective compared to existing implementation (memory and speed wise).
// Turns out calling sync.Map for every context with additional data structures that comes with dependency graph
// is more expensive compared to calling sync.Map for every PerContext service without additional data structures.
package tinysl

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"
)

type locatorRecordDependency struct {
	serviceType string
	id          int
}

type locatorRecord struct {
	dependencies []*locatorRecordDependency
	record
}

func newLocator(ctx context.Context, constructorsByType map[string]*locatorRecord) ServiceLocator {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	singletonsCleanupCh := make(chan cleanupNodeUpdate)
	perContextCleanupCh := make(chan cleanupRecord)
	var wg sync.WaitGroup

	singletons := make([]*locatorRecord, 0)
	perContexts := make([]*locatorRecord, 0)

	perCtxIDs := make([]int, 0)
	for _, rec := range constructorsByType {
		switch rec.lifetime {
		case Singleton:
			singletons = append(singletons, rec)
		case PerContext:
			perCtxIDs = append(perCtxIDs, rec.id)
			perContexts = append(perContexts, rec)
		}
	}
	go singletonCleanupWorker(ctx, cancel, buildCleanupNodes(singletons), singletonsCleanupCh, &wg)

	wg.Add(1)
	go perContextCleanupWorker(
		ctx,
		func(ctx context.Context) cleanupNode {
			n := buildCleanupNodes(perContexts)

			return n
		},
		perContextCleanupCh,
		&wg,
	)

	constructorsByID := make(map[int]*locatorRecord, len(constructorsByType))
	for _, rec := range constructorsByType {
		constructorsByID[rec.id] = rec
	}

	return &locator{
		constructorsByType:  constructorsByType,
		constructorsById:    constructorsByID,
		perContext:          newContextInstances(perCtxIDs),
		singletonsCleanupCh: singletonsCleanupCh,
		perContextCleanUpCh: perContextCleanupCh,
	}
}

type locator struct {
	err                 error
	perContext          *contextInstances
	constructorsByType  map[string]*locatorRecord
	constructorsById    map[int]*locatorRecord
	singletonsCleanupCh chan<- cleanupNodeUpdate
	perContextCleanUpCh chan<- cleanupRecord
	sMu                 sync.Map
	singletons          sync.Map
	errRMu              sync.RWMutex
}

func (l *locator) EnsureAvailable(serviceName string) {
	for key := range l.constructorsByType {
		if key == serviceName {
			return
		}
	}

	l.errRMu.Lock()
	l.err = newConstructorNotFoundError(serviceName)
	l.errRMu.Unlock()
}

func (l *locator) Err() error {
	l.errRMu.RLock()
	defer l.errRMu.RUnlock()

	return l.err
}

func (l *locator) Get(ctx context.Context, serviceName string) (any, error) {
	record, ok := l.constructorsByType[serviceName]

	if !ok {
		return nil, newConstructorNotFoundError(serviceName)
	}

	switch record.lifetime {
	case Singleton:
		return l.getSingleton(ctx, record)
	case PerContext:
		return l.getPerContext(ctx, record, serviceName)
	case Transient:
		s, _, err := l.build(ctx, record)
		return s, err
	default:
		return nil, fmt.Errorf(
			"broken record %s: %w",
			record.typeName,
			LifetimeUnsupportedError(record.lifetime.String()))
	}
}

func (l *locator) get(ctx context.Context, id int, serviceName string) (any, error) {
	record, ok := l.constructorsById[id]

	if !ok {
		return nil, newConstructorNotFoundError(serviceName)
	}

	switch record.lifetime {
	case Singleton:
		return l.getSingleton(ctx, record)
	case PerContext:
		return l.getPerContext(ctx, record, serviceName)
	case Transient:
		s, _, err := l.build(ctx, record)
		return s, err
	default:
		return nil, fmt.Errorf(
			"broken record %s: %w",
			record.typeName,
			LifetimeUnsupportedError(record.lifetime.String()))

	}
}

func (l *locator) build(ctx context.Context, record *locatorRecord) (service any, cleanups Cleanup, err error) {
	defer func() {
		if rp := recover(); rp != nil {
			err = newServiceBuilderError(
				newConstructorError(fmt.Errorf("recovered from panic: %v", rp)),
				record.lifetime,
				record.typeName,
			)
		}
	}()

	constructor := record.constructor
	fn := reflect.ValueOf(constructor)
	args := reflectValuesPool.Get().([]reflect.Value)
	defer func() {
		args = args[:0]
		reflectValuesPool.Put(args)
	}()

	for i, dep := range record.dependencies {
		if i == 0 && dep.id == 0 {
			args = append(args, reflect.ValueOf(ctx))
			continue
		}

		service, err := l.get(ctx, dep.id, dep.serviceType)
		if err != nil {
			return nil, nil, err
		}

		args = append(args, reflect.ValueOf(service))
	}

	values := fn.Call(args)

	if record.constructorType == onlyService && len(values) != 1 ||
		record.constructorType == withError && len(values) != 2 ||
		record.constructorType == withErrorAndCleanUp && len(values) != 3 {
		return nil, nil, newServiceBuilderError(
			newConstructorError(newUnexpectedResultError(values)),
			record.lifetime,
			record.typeName,
		)
	}

	switch record.constructorType {
	case onlyService:
		service := values[0].Interface()
		return service, func() {}, nil
	case withError:
		serviceV, errV := values[0], values[1]
		if err, ok := (errV.Interface()).(error); ok && err != nil {
			return nil, nil, newServiceBuilderError(
				newConstructorError(err),
				record.lifetime,
				record.typeName,
			)
		}

		service := serviceV.Interface()

		return service, func() {}, nil
	case withErrorAndCleanUp:
		serviceV, cleanUpV, errV := values[0], values[1], values[2]
		if err, ok := (errV.Interface()).(error); ok && err != nil {
			return nil, nil, newServiceBuilderError(
				newConstructorError(err),
				record.lifetime,
				record.typeName,
			)
		}

		service := serviceV.Interface()
		cleanUp := cleanUpV.Interface()

		return service, cleanUp.(func()), nil
	default:
		return nil, nil, newServiceBuilderError(
			newConstructorUnsupportedError(
				fn.Type(),
				record.lifetime,
			),
			record.lifetime,
			record.typeName,
		)
	}
}

func (l *locator) getSingleton(ctx context.Context, record *locatorRecord) (any, error) {
	mu, ok := l.sMu.LoadOrStore(record.id, new(sync.Mutex))

	mu.(*sync.Mutex).Lock()
	defer mu.(*sync.Mutex).Unlock()

	servicePtr, ok := l.singletons.Load(record.id)
	if ok {
		return *servicePtr.(*any), nil
	}

	service, cleanUp, err := l.build(ctx, record)
	if err != nil {
		return nil, err
	}

	if record.constructorType == withErrorAndCleanUp {
		go func() {
			l.singletonsCleanupCh <- cleanupNodeUpdate{
				id: record.id,
				fn: func() {
					cleanUp()
					l.sMu.Delete(record.id)
				},
			}
		}()
	}

	l.singletons.Store(record.id, &service)

	return service, nil
}

func (l *locator) getPerContext(ctx context.Context, record *locatorRecord, serviceName string) (any, error) {
	if ctx == nil {
		return nil, newServiceBuilderError(ErrNilContext, record.lifetime, serviceName)
	}

	if err := ctx.Err(); err != nil {
		return nil, newServiceBuilderError(err, record.lifetime, serviceName)
	}

	ctxKey := reflect.ValueOf(ctx).Pointer()
	scope, ok := l.perContext.get(ctxKey, record.id)

	scope.lock()
	defer scope.unlock()

	if !scope.empty() {
		return *scope.value, nil
	}

	service, cleanUp, err := l.build(ctx, record)
	if err != nil {
		return nil, err
	}

	switch {
	case !ok && record.constructorType == withErrorAndCleanUp:
		go func() {
			l.perContextCleanUpCh <- cleanupRecord{
				ctx: ctx,
				cleanupNodeUpdate: cleanupNodeUpdate{
					id: record.id,
					fn: func() { cleanUp() },
				},
			}
			l.perContextCleanUpCh <- cleanupRecord{
				ctx: ctx,
				cleanupNodeUpdate: cleanupNodeUpdate{
					fn: func() { l.perContext.delete(ctxKey) },
				},
			}
		}()
	case !ok:
		go func() {
			l.perContextCleanUpCh <- cleanupRecord{
				ctx: ctx,
				cleanupNodeUpdate: cleanupNodeUpdate{
					fn: func() { l.perContext.delete(ctxKey) },
				},
			}
		}()
	case record.constructorType == withErrorAndCleanUp:
		go func() {
			l.perContextCleanUpCh <- cleanupRecord{
				ctx: ctx,
				cleanupNodeUpdate: cleanupNodeUpdate{
					id: record.id,
					fn: cleanUp,
				},
			}
		}()
	}

	scope.value = &service

	return service, nil
}
