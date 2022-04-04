package tinysl

import (
	"context"
	"reflect"
	"sync"

	"github.com/pkg/errors"
)

var _ ServiceLocator = new(locator)

func newLocator(constructors map[string]record) ServiceLocator {
	return &locator{
		constructors: constructors,
		singletons:   newInstances(),
		perContext:   newContextInstances(),
		sMu:          make(map[string]*sync.Mutex),
		pcMu:         make(map[context.Context]map[string]*sync.Mutex),
	}
}

type locator struct {
	sMuMu  sync.Mutex
	pcMuMu sync.Mutex
	sMu    map[string]*sync.Mutex
	pcMu   map[context.Context]map[string]*sync.Mutex

	singletons   *instances
	perContext   *contextInstances
	constructors map[string]record
}

func (l *locator) Get(ctx context.Context, serviceName string) (any, error) {
	record, ok := l.constructors[serviceName]

	if !ok {
		return nil, errors.Errorf(constructorNotFound, serviceName)
	}

	switch record.lifetime {
	case Singleton:
		return l.getSingleton(ctx, record, serviceName)
	case PerContext:
		return l.getPerContext(ctx, record, serviceName)
	case Transient:
		return l.get(ctx, record)
	default:
		panic(errors.Errorf("broken record %s: unexpected Lifetime %s", record.typeName, record.lifetime))
	}
}

func (l *locator) get(ctx context.Context, record record) (any, error) {
	constructor := record.constructor
	fn := reflect.ValueOf(constructor)
	args := make([]reflect.Value, 0, 1)

	for i, dep := range record.dependencies {
		if i == 0 && dep == contextDepName {
			args = append(args, reflect.ValueOf(ctx))
			continue
		}

		service, err := l.Get(ctx, dep)

		if err != nil {
			return nil, err
		}

		args = append(args, reflect.ValueOf(service))
	}

	values := fn.Call(args)

	if len(values) != 2 {
		return nil, errors.Errorf(constructorReturnedBadResult, constructor, values)
	}

	serviceV, errV := values[0], values[1]
	if err, ok := (errV.Interface()).(error); ok && err != nil {
		return nil, errors.Wrapf(err, constructorReturnedError, constructor)
	}

	service := serviceV.Interface()

	return service, nil
}

func (l *locator) getSingleton(ctx context.Context, record record, serviceName string) (any, error) {
	l.sMuMu.Lock()

	if _, ok := l.sMu[serviceName]; !ok {
		l.sMu[serviceName] = new(sync.Mutex)
	}

	mu := l.sMu[serviceName]

	l.sMuMu.Unlock()

	mu.Lock()
	defer mu.Unlock()

	service, ok := l.singletons.get(serviceName)
	if ok {
		return service, nil
	}

	service, err := l.get(ctx, record)
	if err != nil {
		return nil, err
	}

	l.singletons.set(serviceName, service)

	return service, nil
}

func (l *locator) getPerContext(ctx context.Context, record record, serviceName string) (any, error) {
	if ctx == nil {
		return nil, errors.Wrapf(errors.New("got nil context"), cannotBuildService, record.lifetime, serviceName)
	}

	if err := ctx.Err(); err != nil {
		return nil, errors.Wrapf(err, cannotBuildService, record.lifetime, serviceName)
	}

	l.pcMuMu.Lock()

	if _, ok := l.pcMu[ctx]; !ok {
		l.pcMu[ctx] = make(map[string]*sync.Mutex)

		go func() {
			<-ctx.Done()
			l.pcMuMu.Lock()

			delete(l.pcMu, ctx)
			l.perContext.delete(ctx)

			l.pcMuMu.Unlock()
		}()
	}

	if _, ok := l.pcMu[ctx][serviceName]; !ok {
		l.pcMu[ctx][serviceName] = new(sync.Mutex)
	}

	mu := l.pcMu[ctx][serviceName]

	l.pcMuMu.Unlock()

	mu.Lock()
	defer mu.Unlock()

	service, ok := l.perContext.get(ctx, serviceName)
	if ok {
		return service, nil
	}

	service, err := l.get(ctx, record)
	if err != nil {
		return nil, err
	}

	l.perContext.set(ctx, serviceName, service)

	return service, nil
}
