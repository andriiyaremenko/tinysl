package tinysl

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"syscall"
)

var reflectValuesPool = sync.Pool{
	New: func() any {
		val := make([]reflect.Value, 0, 1)
		return &val
	},
}

type locatorRecord struct {
	dependencies []*locatorRecord
	record
}

func newLocator(ctx context.Context, constructorsByType map[string]*locatorRecord, numS, numP int32) ServiceLocator {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	singletonsCleanupCh := make(chan cleanupNodeUpdate)

	singletons := make([]*locatorRecord, numS)
	perContexts := make([]*locatorRecord, numP)

	for _, rec := range constructorsByType {
		switch rec.lifetime {
		case Singleton:
			singletons[rec.id] = rec
		case PerContext:
			perContexts[rec.id] = rec
		}
	}

	go singletonCleanupWorker(ctx, cancel, buildCleanupNodes(singletons), singletonsCleanupCh)

	cleanupNodeBuilder := func() *cleanupNode {
		return buildCleanupNodes(perContexts)
	}

	singletonsServices := make([]*serviceScope, numS)
	for i := range singletonsServices {
		singletonsServices[i] = &serviceScope{}
	}

	return &locator{
		constructorsByType:  constructorsByType,
		perContext:          newContextInstances(numP, cleanupNodeBuilder),
		singletonsCleanupCh: singletonsCleanupCh,
		singletons:          singletonsServices,
	}
}

type locator struct {
	err                 atomic.Pointer[error]
	perContext          *contextInstances
	constructorsByType  map[string]*locatorRecord
	singletonsCleanupCh chan<- cleanupNodeUpdate
	singletons          []*serviceScope
}

func (l *locator) EnsureAvailable(serviceName string) {
	for key := range l.constructorsByType {
		if key == serviceName {
			return
		}
	}

	err := newConstructorNotFoundError(serviceName)
	l.err.Store(&err)
}

func (l *locator) Err() error {
	if err := l.err.Load(); err != nil {
		return *err
	}
	return nil
}

func (l *locator) Get(ctx context.Context, serviceName string) (service any, err error) {
	record, ok := l.constructorsByType[serviceName]

	if !ok {
		return nil, newConstructorNotFoundError(serviceName)
	}

	defer func() {
		if rp := recover(); rp != nil {
			err = newServiceBuilderError(
				newConstructorError(newRecoveredError(rp, debug.Stack())),
				record.lifetime,
				record.typeName,
			)
		}
	}()

	return l.get(ctx, record)
}

func (l *locator) get(ctx context.Context, record *locatorRecord) (any, error) {
	switch record.lifetime {
	case Singleton:
		return l.getSingleton(ctx, record)
	case PerContext:
		return l.getPerContext(ctx, record)
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

func (l *locator) build(ctx context.Context, record *locatorRecord) (any, Cleanup, error) {
	constructor := record.constructor
	fn := reflect.ValueOf(constructor)
	argsPtr := reflectValuesPool.Get().(*[]reflect.Value)
	args := *argsPtr
	defer func() {
		args = args[:0]
		reflectValuesPool.Put(argsPtr)
	}()

	for i, dep := range record.dependencies {
		if i == 0 && dep.id == -1 {
			args = append(args, reflect.ValueOf(ctx))
			continue
		}

		service, err := l.get(ctx, dep)
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
	scope := l.singletons[record.id]

	scope.lock()
	defer scope.unlock()

	if !scope.empty() {
		return *scope.value, nil
	}

	service, cleanUp, err := l.build(ctx, record)
	if err != nil {
		return nil, err
	}

	scope.value = &service

	if record.constructorType == withErrorAndCleanUp {
		go func() {
			l.singletonsCleanupCh <- cleanupNodeUpdate{
				id: record.id,
				fn: func() {
					cleanUp()
					l.singletons[record.id].lock()
					l.singletons[record.id].value = nil
					l.singletons[record.id].unlock()
				},
			}
		}()
	}

	return service, nil
}

func (l *locator) getPerContext(ctx context.Context, record *locatorRecord) (any, error) {
	if ctx == nil {
		return nil, newServiceBuilderError(ErrNilContext, record.lifetime, record.typeName)
	}

	if err := ctx.Err(); err != nil {
		return nil, newServiceBuilderError(err, record.lifetime, record.typeName)
	}

	scope, cleanupNode := l.perContext.get(ctx, record.id)

	scope.lock()
	defer scope.unlock()

	if !scope.empty() {
		return *scope.value, nil
	}

	service, cleanUp, err := l.build(ctx, record)
	if err != nil {
		return nil, err
	}

	scope.value = &service

	if record.constructorType == withErrorAndCleanUp {
		cleanupNode.updateCleanupNode(record.id, cleanUp)
	}

	return service, nil
}
