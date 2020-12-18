package tinysl

import (
	"context"
	"reflect"
	"sync"

	"github.com/pkg/errors"
)

const (
	errAddTransientOrSingletonText string = "constructor should be of type func() (T, error) for Transient and Singleton, got %s"
	errAddPerRequestText           string = "constructor should be of type func(context.Context) (T, error) for PerContext, got %s"
)

func New() ServiceLocator {
	return &locator{
		singletons:   make(map[string]interface{}),
		perContext:   make(map[context.Context]map[string]interface{}),
		constructors: make(map[string]record)}
}

var errorInterface = reflect.TypeOf((*error)(nil)).Elem()
var contextInterface = reflect.TypeOf((*context.Context)(nil)).Elem()

type record struct {
	lifetime    lifetime
	constructor interface{}
}

type locator struct {
	singletonsMu     sync.Mutex
	perContextMu     sync.Mutex
	constructorsRWMu sync.RWMutex

	singletons   map[string]interface{}
	perContext   map[context.Context]map[string]interface{}
	constructors map[string]record
}

func (l *locator) sealed() {}

func (l *locator) Add(lifetime lifetime, constructor interface{}) error {
	errAddText := errAddTransientOrSingletonText
	if lifetime == PerContext {
		errAddText = errAddPerRequestText
	}

	t := reflect.TypeOf(constructor)
	if t.Kind() != reflect.Func {
		return errors.Errorf(errAddText, t)
	}

	numIn := t.NumIn()

	if lifetime == PerContext && numIn > 1 ||
		lifetime != PerContext && numIn != 0 {
		return errors.Errorf(errAddText, t)
	}

	if numIn == 1 && !t.In(0).Implements(contextInterface) {
		return errors.Errorf(errAddText, t)
	}

	numOut := t.NumOut()
	if numOut != 2 {
		return errors.Errorf(errAddText, t)
	}

	errType := t.Out(1)
	if !errType.Implements(errorInterface) {
		return errors.Errorf(errAddText, t)
	}

	l.constructorsRWMu.RLock()

	serviceType := t.Out(0).String()
	if v, ok := l.constructors[serviceType]; ok {
		l.constructorsRWMu.RUnlock()

		return errors.Errorf("ServiceLocator has already registered constructor for %s: %T", serviceType, v)
	}

	l.constructorsRWMu.RUnlock()
	l.constructorsRWMu.Lock()

	l.constructors[serviceType] = record{lifetime: lifetime, constructor: constructor}

	l.constructorsRWMu.Unlock()

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

	l.constructorsRWMu.RLock()

	record, ok := l.constructors[serviceName]

	l.constructorsRWMu.RUnlock()

	if !ok {
		return nil, errors.Errorf("constructor for %s not found", serviceType)
	}

	switch record.lifetime {
	case PerContext:
		l.perContextMu.Lock()
		defer l.perContextMu.Unlock()
	case Singleton:
		l.singletonsMu.Lock()
		defer l.singletonsMu.Unlock()
	}

	if record.lifetime == Singleton {
		if service, ok := l.singletons[serviceName]; ok {
			return service, nil
		}
	}

	if record.lifetime == PerContext {
		if err := ctx.Err(); err != nil {
			return nil, errors.Wrapf(err, "PerContext service %s cannot be served for cancelled context", serviceType)
		}

		if l.perContext[ctx] == nil {
			l.perContext[ctx] = make(map[string]interface{})

			go func() {
				<-ctx.Done()

				l.perContextMu.Lock()
				delete(l.perContext, ctx)
				l.perContextMu.Unlock()
			}()
		}

		if service, ok := l.perContext[ctx][serviceName]; ok {
			return service, nil
		}
	}

	constructor := record.constructor
	fn := reflect.ValueOf(constructor)
	args := make([]reflect.Value, 0, 1)

	if reflect.TypeOf(constructor).NumIn() == 1 {
		args = append(args, reflect.ValueOf(ctx))
	}

	values := fn.Call(args)

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
		l.singletons[serviceName] = service
	case PerContext:
		l.perContext[ctx][serviceName] = service
	}

	return service, nil
}
