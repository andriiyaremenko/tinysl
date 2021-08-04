package tinysl

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/pkg/errors"
)

const (
	contextDepName = "context.Context"

	templateConstructor                  string = "constructor should be of type %s for %s, got %s"
	templateConstructorNotFound          string = "constructor for %s not found"
	templateConstructorReturnedError     string = "constructor %T returned an error"
	templateConstructorReturnedBadResult string = "constructor %T returned an unexpected result: %v"
	templateVariadicConstructor          string = "variadic function as constructor is not supported: got %s"
	templateDuplicateConstructor         string = "ServiceLocator has already registered constructor for %s: %T"
	templateNotAPointer                  string = "service type should be pointer type, got: %s"
	templateMissingDependency            string = "%s has unregistered dependency: %s"
	templateCircularDependency           string = "circular dependency in %T: %s depends on %s"
	templatePerContextNilContext         string = "PerContext service %s cannot be served for nil context"
	templatePerContextCancelledContext   string = "PerContext service %s cannot be served for cancelled context"

	constructorTypeStr            string = "func(T1, T2, ...) (T, error)"
	constructorWithContextTypeStr string = "func(context.Context, T1, T2, ...) (T, error)"

	singletonPossibleConstructor  string = constructorTypeStr
	perContextPossibleConstructor string = constructorTypeStr + " | " + constructorWithContextTypeStr
	transientPossibleConstructor  string = constructorTypeStr + " | " + constructorWithContextTypeStr
)

var errorInterface = reflect.TypeOf((*error)(nil)).Elem()
var contextInterface = reflect.TypeOf((*context.Context)(nil)).Elem()

var lifetimeLookup = map[lifetime]string{
	Singleton:  "Singleton",
	PerContext: "PerContext",
	Transient:  "Transient"}

// returns new ServiceLocator.
func New() ServiceLocator {
	return &locator{
		singletonsMs: make(map[string]*sync.Mutex),
		perContextMs: make(map[string]*sync.Mutex),

		singletons:   make(map[string]interface{}),
		perContext:   make(map[context.Context]map[string]interface{}),
		constructors: make(map[string]record)}
}

type record struct {
	lifetime     lifetime
	constructor  interface{}
	dependencies []string
	typeName     string
}

type locator struct {
	singletonsMs    map[string]*sync.Mutex
	perContextMs    map[string]*sync.Mutex
	perContextM     sync.Mutex
	constructorsRWM sync.RWMutex

	singletons   map[string]interface{}
	perContext   map[context.Context]map[string]interface{}
	constructors map[string]record
}

func (l *locator) sealed() {}

func (l *locator) Add(lifetime lifetime, constructor interface{}) error {
	var errAddText string

	t := reflect.TypeOf(constructor)

	switch lifetime {
	case Singleton:
		errAddText = fmt.Sprintf(
			templateConstructor,
			singletonPossibleConstructor,
			lifetimeLookup[lifetime],
			t)
	case PerContext:
		errAddText = fmt.Sprintf(
			templateConstructor,
			perContextPossibleConstructor,
			lifetimeLookup[lifetime],
			t)
	case Transient:
		errAddText = fmt.Sprintf(
			templateConstructor,
			transientPossibleConstructor,
			lifetimeLookup[lifetime],
			t)
	}

	if t.Kind() != reflect.Func {
		return errors.New(errAddText)
	}

	if t.IsVariadic() {
		return errors.Errorf(templateVariadicConstructor, t)
	}

	numIn := t.NumIn()

	// Singleton cannot be based on any context, but PerContext and Transient can
	if lifetime == Singleton &&
		numIn > 0 &&
		t.In(0).Implements(contextInterface) {
		return errors.New(errAddText)
	}

	numOut := t.NumOut()
	if numOut != 2 {
		return errors.New(errAddText)
	}

	errType := t.Out(1)
	if !errType.Implements(errorInterface) {
		return errors.New(errAddText)
	}

	l.constructorsRWM.RLock()

	serviceType := t.Out(0).String()
	if v, ok := l.constructors[serviceType]; ok {
		l.constructorsRWM.RUnlock()

		return errors.Errorf(templateDuplicateConstructor, serviceType, v)
	}

	l.constructorsRWM.RUnlock()
	l.constructorsRWM.Lock()

	r := record{lifetime: lifetime, constructor: constructor, typeName: serviceType}

	for i := 0; i < numIn; i++ {
		argT := t.In(i)
		if i > 0 && argT.Implements(contextInterface) {
			return errors.New(errAddText)
		}

		if argT.Implements(contextInterface) {
			r.dependencies = append(r.dependencies, contextDepName)
			continue
		}

		r.dependencies = append(r.dependencies, argT.String())
	}

	l.constructors[serviceType] = r

	switch lifetime {
	case Singleton:
		l.singletonsMs[serviceType] = new(sync.Mutex)
	case PerContext:
		l.perContextMs[serviceType] = new(sync.Mutex)
	}

	l.constructorsRWM.Unlock()

	return nil
}

func (l *locator) Get(ctx context.Context, servicePrt interface{}) (interface{}, error) {
	serviceType := reflect.TypeOf(servicePrt)
	if serviceType.Kind() != reflect.Ptr {
		return nil, errors.Errorf(templateNotAPointer, serviceType)
	}

	serviceName := serviceType.Elem().String()

	if l.constructors == nil {
		return nil, errors.Errorf(templateConstructorNotFound, serviceName)
	}

	return l.get(ctx, serviceName, serviceName)
}

func (l *locator) CanResolveDependencies() error {
	l.constructorsRWM.RLock()
	defer l.constructorsRWM.RUnlock()

	for _, record := range l.constructors {
		if err := l.canResolveDependencies(record); err != nil {
			return err
		}
	}

	return nil
}

func (l *locator) canResolveDependencies(record record, dependentServiceNames ...string) error {
	dependentServiceNames = append(dependentServiceNames, record.typeName)
	for _, dependency := range record.dependencies {
		if dependency == contextDepName {
			continue
		}

		r, ok := l.constructors[dependency]
		if !ok {
			return errors.Errorf(
				templateMissingDependency,
				record.typeName,
				fmt.Sprintf(templateConstructorNotFound, dependency),
			)
		}

		if hasServiceName(dependency, dependentServiceNames) {
			return errors.Errorf(
				templateCircularDependency,
				record.constructor,
				record.typeName,
				dependency)
		}

		if err := l.canResolveDependencies(r, dependentServiceNames...); err != nil {
			return err
		}
	}

	return nil
}

func (l *locator) get(
	ctx context.Context,
	serviceName string,
	initialServiceNames ...string,
) (interface{}, error) {
	l.constructorsRWM.RLock()

	record, ok := l.constructors[serviceName]

	l.constructorsRWM.RUnlock()

	if !ok {
		return nil, errors.Errorf(templateConstructorNotFound, serviceName)
	}

	switch record.lifetime {
	case PerContext:
		l.perContextMs[serviceName].Lock()
		defer l.perContextMs[serviceName].Unlock()
	case Singleton:
		l.singletonsMs[serviceName].Lock()
		defer l.singletonsMs[serviceName].Unlock()
	}

	if record.lifetime == Singleton {
		if service, ok := l.singletons[serviceName]; ok {
			return service, nil
		}
	}

	if record.lifetime == PerContext {
		if ctx == nil {
			return nil, errors.Errorf(templatePerContextNilContext, serviceName)
		}

		if err := ctx.Err(); err != nil {
			return nil, errors.Wrapf(err, templatePerContextCancelledContext, serviceName)
		}

		l.perContextM.Lock()
		if l.perContext[ctx] == nil {
			l.perContext[ctx] = make(map[string]interface{})

			go func() {
				<-ctx.Done()

				l.perContextM.Lock()
				delete(l.perContext, ctx)
				l.perContextM.Unlock()
			}()
		}

		if service, ok := l.perContext[ctx][serviceName]; ok {
			l.perContextM.Unlock()

			return service, nil
		}

		l.perContextM.Unlock()
	}

	constructor := record.constructor
	fn := reflect.ValueOf(constructor)
	args := make([]reflect.Value, 0, 1)

	for i, dep := range record.dependencies {
		if hasServiceName(dep, initialServiceNames) {
			return nil, errors.Errorf(
				templateCircularDependency,
				constructor,
				record.typeName,
				dep)
		}

		if i == 0 && dep == contextDepName {
			args = append(args, reflect.ValueOf(ctx))
			continue
		}

		initialServiceNames = append(initialServiceNames, serviceName)
		service, err := l.get(ctx, dep, initialServiceNames...)

		if err != nil {
			return nil, err
		}

		args = append(args, reflect.ValueOf(service))
	}

	values := fn.Call(args)

	if len(values) != 2 {
		return nil, errors.Errorf(templateConstructorReturnedBadResult, constructor, values)
	}

	serviceV, errV := values[0], values[1]
	if err, ok := (errV.Interface()).(error); ok && err != nil {
		return nil, errors.Wrapf(err, templateConstructorReturnedError, constructor)
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

func hasServiceName(name string, serviceNames []string) bool {
	for _, serviceName := range serviceNames {
		if serviceName == name {
			return true
		}
	}

	return false
}
