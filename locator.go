package tinysl

import (
	"context"
	"reflect"
	"sync"

	"github.com/pkg/errors"
)

func New() ServiceLocator {
	return &locator{
		singletons:   make(map[string]interface{}),
		perContext:   make(map[context.Context]map[string]interface{}),
		constructors: make(map[string]record)}
}

var errorInterface = reflect.TypeOf((*error)(nil)).Elem()

type record struct {
	lifetime    lifetime
	constructor interface{}
}

type locator struct {
	singletonsRWM   sync.RWMutex
	perContextRWM   sync.RWMutex
	constructorsRWM sync.RWMutex

	singletons   map[string]interface{}
	perContext   map[context.Context]map[string]interface{}
	constructors map[string]record
}

func (l *locator) sealed() {}

func (l *locator) Add(lifetime lifetime, constructor interface{}) error {
	t := reflect.TypeOf(constructor)
	if t.Kind() != reflect.Func {
		return errors.Errorf("%s constructor should be a function", t)
	}

	numIn := t.NumIn()
	if numIn != 0 {
		return errors.Errorf("%s constructor should have 0 arguments, got %d", t, numIn)
	}

	numOut := t.NumOut()
	if numOut != 2 {
		return errors.Errorf("%s constructor should have 2 returning parameters, got %d", t, numOut)
	}

	errType := t.Out(1)
	if !errType.Implements(errorInterface) {
		return errors.Errorf("2nd returning type of constructor should be error got %s", errType)
	}

	l.constructorsRWM.RLock()

	serviceType := t.Out(0).String()
	if v, ok := l.constructors[serviceType]; ok {
		l.constructorsRWM.RUnlock()

		return errors.Errorf("di has already registered constructor for %s: %T", serviceType, v)
	}

	l.constructorsRWM.RUnlock()
	l.constructorsRWM.Lock()

	l.constructors[serviceType] = record{lifetime: lifetime, constructor: constructor}

	l.constructorsRWM.Unlock()

	return nil
}

func (l *locator) Get(ctx context.Context, serviceType reflect.Type) (interface{}, error) {
	if serviceType.Kind() != reflect.Ptr {
		return nil, errors.Errorf("service type should be pointer type, got: %s", serviceType)
	}
	serviceName := serviceType.Elem().String()

	if l.constructors == nil {
		return nil, errors.Errorf("constructor for %s not found", serviceName)
	}

	l.constructorsRWM.RLock()

	record, ok := l.constructors[serviceName]

	l.constructorsRWM.RUnlock()

	if !ok {
		return nil, errors.Errorf("constructor for %s not found", serviceType)
	}

	if record.lifetime == Singleton {
		l.singletonsRWM.RLock()

		if service, ok := l.singletons[serviceName]; ok {
			l.singletonsRWM.RUnlock()

			return service, nil
		}

		l.singletonsRWM.RUnlock()
	}

	l.perContextRWM.RLock()

	if l.perContext[ctx] == nil {
		l.perContextRWM.RUnlock()
		l.perContextRWM.Lock()

		l.perContext[ctx] = make(map[string]interface{})

		l.perContextRWM.Unlock()
		l.perContextRWM.RLock()

		go func() {
			<-ctx.Done()

			l.perContextRWM.Lock()

			delete(l.perContext, ctx)

			l.perContextRWM.Unlock()
		}()
	}

	l.perContextRWM.RUnlock()

	if record.lifetime == PerContext {
		if err := ctx.Err(); err != nil {
			return nil, errors.Wrapf(err, "PerContext service %s cannot be served for cancelled context", serviceType)
		}

		l.perContextRWM.RLock()

		if service, ok := l.perContext[ctx][serviceName]; ok {
			l.perContextRWM.RUnlock()

			return service, nil
		}

		l.perContextRWM.RUnlock()
	}

	constructor := record.constructor
	fn := reflect.ValueOf(constructor)
	values := fn.Call(make([]reflect.Value, 0))

	if len(values) != 2 {
		return nil, errors.Errorf("constructor %T returned an unexpected result: %v", constructor, values)
	}

	serviceV, errV := values[0], values[1]
	if err, ok := (errV.Interface()).(error); ok && err != nil {
		return nil, errors.Wrapf(err, "constructor %T returned an error", constructor)
	}

	service := serviceV.Interface()

	switch record.lifetime {
	case Singleton:
		l.singletonsRWM.Lock()

		l.singletons[serviceName] = service

		l.singletonsRWM.Unlock()
	case PerContext:
		l.perContextRWM.Lock()

		l.perContext[ctx][serviceName] = service

		l.perContextRWM.Unlock()
	}

	return service, nil
}
