package tinysl

import (
	"reflect"
	"sync"
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
	return &container{constructors: make(map[string]record)}
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

	if lifetime != Singleton &&
		lifetime != PerContext &&
		lifetime != Transient {
		c.err = LifetimeUnsupportedError(lifetime)

		return c
	}

	// Check if constructor returns Constructor type
	construct, ok := constructor.(func() (propertyFiller, error))
	if ok {
		constructor, err := construct()
		if err != nil {
			c.err = err

			return c
		}

		t := constructor.Type
		serviceType := t.String()
		r := record{
			typeName:     serviceType,
			lifetime:     lifetime,
			dependencies: constructor.Dependencies,
			constructor:  constructor.NewInstance,
		}

		c.constructorsRWM.Lock()
		defer c.constructorsRWM.Unlock()

		if _, ok := c.constructors[serviceType]; ok {
			c.err = newBadConstructorError(ErrDuplicateConstructor, t)

			return c
		}

		c.constructors[serviceType] = r

		return c
	}

	// Regular constructor
	t := reflect.TypeOf(constructor)

	if t.Kind() != reflect.Func {
		c.err = newConstructorUnsupportedError(t, lifetime)

		return c
	}

	if t.IsVariadic() {
		c.err = newBadConstructorError(ErrVariadicConstructor, t)

		return c
	}

	numIn := t.NumIn()

	// Singleton cannot be based on any context, but PerContext and Transient can
	if lifetime == Singleton && numIn > 0 && t.In(0).Implements(contextInterface) {
		c.err = newConstructorUnsupportedError(t, lifetime)

		return c
	}

	numOut := t.NumOut()
	if numOut != 2 {
		c.err = newConstructorUnsupportedError(t, lifetime)

		return c
	}

	errType := t.Out(1)
	if !errType.Implements(errorInterface) {
		c.err = newConstructorUnsupportedError(t, lifetime)

		return c
	}

	c.constructorsRWM.Lock()
	defer c.constructorsRWM.Unlock()

	serviceType := t.Out(0).String()
	if _, ok := c.constructors[serviceType]; ok {
		c.err = newBadConstructorError(ErrDuplicateConstructor, t)

		return c
	}

	r := record{lifetime: lifetime, constructor: constructor, typeName: serviceType}

	for i := 0; i < numIn; i++ {
		argT := t.In(i)
		if i > 0 && argT.Implements(contextInterface) {
			c.err = newConstructorUnsupportedError(t, lifetime)

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
			return NewServiceBuilderError(
				NewConstructorNotFoundError(dependency),
				record.lifetime,
				record.typeName,
			)
		}

		for _, serviceName := range dependentServiceNames {
			if serviceName == dependency {
				return NewServiceBuilderError(
					NewCircularDependencyError(record.constructor, dependency),
					record.lifetime,
					record.typeName,
				)
			}
		}

		if err := c.canResolveDependencies(r, dependentServiceNames...); err != nil {
			return err
		}
	}

	return nil
}
