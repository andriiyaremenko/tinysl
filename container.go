package tinysl

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/exp/slices"
)

var _ ServiceLocator = new(container)
var _ Container = new(container)

type record struct {
	lifetime     Lifetime
	constructor  any
	dependencies []string
	typeName     string
}

func newContainer() *container {
	return &container{
		instantiated: true,
		singletons:   newInstances(),
		perContext:   newContextInstances(),
		constructors: make(map[string]record),
	}
}

type container struct {
	constructorsRWM sync.RWMutex

	err          error
	instantiated bool

	singletons   *instances
	perContext   *contextInstances
	constructors map[string]record
}

func (sl *container) Get(ctx context.Context, serviceName string) (any, error) {
	sl.constructorsRWM.RLock()

	record, ok := sl.constructors[serviceName]

	sl.constructorsRWM.RUnlock()

	if !ok {
		return nil, errors.Errorf(constructorNotFound, serviceName)
	}

	if record.lifetime == Singleton {
		if service, ok := sl.singletons.get(serviceName); ok {
			return service, nil
		}
	}

	if record.lifetime == PerContext {
		if ctx == nil {
			return nil, errors.Errorf(nilContext, serviceName)
		}

		if err := ctx.Err(); err != nil {
			return nil, errors.Wrapf(err, cancelledContext, serviceName)
		}

		if service, ok := sl.perContext.get(ctx, serviceName); ok {
			return service, nil
		}
	}

	constructor := record.constructor
	fn := reflect.ValueOf(constructor)
	args := make([]reflect.Value, 0, 1)

	for i, dep := range record.dependencies {
		if i == 0 && dep == contextDepName {
			args = append(args, reflect.ValueOf(ctx))
			continue
		}

		service, err := sl.Get(ctx, dep)

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

	switch record.lifetime {
	case Singleton:
		sl.singletons.set(serviceName, service)
	case PerContext:
		sl.perContext.set(ctx, serviceName, service)
	}

	return service, nil
}

func (c *container) Err() error {
	c.constructorsRWM.RLock()
	defer c.constructorsRWM.RUnlock()

	if !c.instantiated {
		return errors.New(containerIsNotInstantiated)
	}

	return c.err
}

func (sl *container) Add(lifetime Lifetime, constructor any) Container {
	if sl.err != nil {
		return sl
	}

	t := reflect.TypeOf(constructor)
	var errAddText string

	switch lifetime {
	case Singleton:
		errAddText = fmt.Sprintf(wrongConstructor, singletonPossibleConstructor, lifetime, t)
	case PerContext:
		errAddText = fmt.Sprintf(wrongConstructor, perContextPossibleConstructor, lifetime, t)
	case Transient:
		errAddText = fmt.Sprintf(wrongConstructor, transientPossibleConstructor, lifetime, t)
	default:
		sl.err = errors.Errorf(unsupportedLifetime, lifetime)

		return sl
	}

	if t.Kind() != reflect.Func {
		sl.err = errors.New(errAddText)

		return sl
	}

	if t.IsVariadic() {
		sl.err = errors.Errorf(variadicConstructorUnsupported, t)

		return sl
	}

	numIn := t.NumIn()

	// Singleton cannot be based on any context, but PerContext and Transient can
	if lifetime == Singleton && numIn > 0 && t.In(0).Implements(contextInterface) {
		sl.err = errors.New(errAddText)

		return sl
	}

	numOut := t.NumOut()
	if numOut != 2 {
		sl.err = errors.New(errAddText)

		return sl
	}

	errType := t.Out(1)
	if !errType.Implements(errorInterface) {
		sl.err = errors.New(errAddText)

		return sl
	}

	sl.constructorsRWM.Lock()
	defer sl.constructorsRWM.Unlock()

	if !sl.instantiated {
		sl = newContainer()
	}

	serviceType := t.Out(0).String()
	if v, ok := sl.constructors[serviceType]; ok {
		sl.err = errors.Errorf(duplicateConstructor, serviceType, v.constructor)

		return sl
	}

	r := record{lifetime: lifetime, constructor: constructor, typeName: serviceType}

	for i := 0; i < numIn; i++ {
		argT := t.In(i)
		if i > 0 && argT.Implements(contextInterface) {
			sl.err = errors.New(errAddText)

			return sl
		}

		if argT.Implements(contextInterface) {
			r.dependencies = append(r.dependencies, contextDepName)
			continue
		}

		r.dependencies = append(r.dependencies, argT.String())
	}

	sl.constructors[serviceType] = r

	return sl
}

func (c *container) ServiceLocator() (ServiceLocator, error) {
	c.constructorsRWM.RLock()
	defer c.constructorsRWM.RUnlock()

	if c.err != nil {
		return nil, c.err
	}

	for _, record := range c.constructors {
		if err := c.canResolveDependencies(record); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (c *container) canResolveDependencies(record record, dependentServiceNames ...string) error {
	dependentServiceNames = append(dependentServiceNames, record.typeName)
	for _, dependency := range record.dependencies {
		if dependency == contextDepName {
			continue
		}

		r, ok := c.constructors[dependency]
		if !ok {
			return errors.Errorf(
				missingDependency,
				record.typeName,
				fmt.Sprintf(constructorNotFound, dependency),
			)
		}

		if slices.Contains(dependentServiceNames, dependency) {
			return errors.Errorf(
				circularDependencyFound,
				record.constructor,
				record.typeName,
				dependency)
		}

		if err := c.canResolveDependencies(r, dependentServiceNames...); err != nil {
			return err
		}
	}

	return nil
}
