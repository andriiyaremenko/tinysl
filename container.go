package tinysl

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/exp/slices"
)

var _ Container = new(container)

// Returns new Container.
func New() Container {
	return newContainer()
}

// Creates new Container, adds constructor and returns newly-created container.
func Add(lifetime Lifetime, constructor any) Container {
	return New().Add(lifetime, constructor)
}

type record struct {
	lifetime     Lifetime
	constructor  any
	dependencies []string
	typeName     string
}

func newContainer() *container {
	return &container{
		constructors: make(map[string]record),
	}
}

type container struct {
	constructorsRWM sync.RWMutex

	err          error
	constructors map[string]record
}

func (c *container) Add(lifetime Lifetime, constructor any) Container {
	if c.err != nil {
		return c
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
		c.err = errors.Errorf(unsupportedLifetime, lifetime)

		return c
	}

	if t.Kind() != reflect.Func {
		c.err = errors.New(errAddText)

		return c
	}

	if t.IsVariadic() {
		c.err = errors.Errorf(variadicConstructorUnsupported, t)

		return c
	}

	numIn := t.NumIn()

	// Singleton cannot be based on any context, but PerContext and Transient can
	if lifetime == Singleton && numIn > 0 && t.In(0).Implements(contextInterface) {
		c.err = errors.New(errAddText)

		return c
	}

	numOut := t.NumOut()
	if numOut != 2 {
		c.err = errors.New(errAddText)

		return c
	}

	errType := t.Out(1)
	if !errType.Implements(errorInterface) {
		c.err = errors.New(errAddText)

		return c
	}

	c.constructorsRWM.Lock()
	defer c.constructorsRWM.Unlock()

	serviceType := t.Out(0).String()
	if v, ok := c.constructors[serviceType]; ok {
		c.err = errors.Errorf(duplicateConstructor, serviceType, v.constructor)

		return c
	}

	r := record{lifetime: lifetime, constructor: constructor, typeName: serviceType}

	for i := 0; i < numIn; i++ {
		argT := t.In(i)
		if i > 0 && argT.Implements(contextInterface) {
			c.err = errors.New(errAddText)

			return c
		}

		if argT.Implements(contextInterface) {
			r.dependencies = append(r.dependencies, contextDepName)
			continue
		}

		r.dependencies = append(r.dependencies, argT.String())
	}

	c.constructors[serviceType] = r

	return c
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

	return newLocator(c.constructors), nil
}

func (c *container) canResolveDependencies(record record, dependentServiceNames ...string) error {
	dependentServiceNames = append(dependentServiceNames, record.typeName)
	for _, dependency := range record.dependencies {
		if dependency == contextDepName {
			continue
		}

		r, ok := c.constructors[dependency]
		if !ok {
			return errors.Wrapf(
				errors.Errorf(constructorNotFound, dependency),
				missingDependency,
				record.typeName,
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
