package tinysl

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

var _ Container = new(container)

type ContainerConfiguration struct {
	Ctx                         context.Context
	SilenceUseSingletonWarnings bool
}

type ContainerOption func(*ContainerConfiguration)

var (
	WithSingletonCleanupContext = func(ctx context.Context) ContainerOption {
		return func(opt *ContainerConfiguration) { opt.Ctx = ctx }
	}

	SilenceUseSingletonWarnings ContainerOption = func(opt *ContainerConfiguration) { opt.SilenceUseSingletonWarnings = true }
)

// Returns new Container.
func New(opts ...ContainerOption) Container {
	conf := ContainerConfiguration{
		Ctx: context.Background(),
	}

	for _, opt := range opts {
		opt(&conf)
	}

	return newContainer(conf.Ctx, conf.SilenceUseSingletonWarnings)
}

// Creates new Container, adds constructor and returns newly-created container.
func Add(lifetime Lifetime, constructor any) Container {
	return New().Add(lifetime, constructor)
}

type constructorType int

const (
	onlyService constructorType = iota
	withError
	withErrorAndCleanUp
)

type record struct {
	constructor      any
	typeName         string
	dependencies     []string
	id               uintptr
	constructorType  constructorType
	lifetime         Lifetime
	dependsOnContext bool
}

func newContainer(ctx context.Context, silenceUseSingletonWarnings bool) *container {
	return &container{constructors: make(map[string]*record), ctx: ctx, ignoreScopeAnalyzerErrors: silenceUseSingletonWarnings}
}

type container struct {
	ctx                       context.Context
	err                       error
	constructors              map[string]*record
	constructorsRWM           sync.RWMutex
	ignoreScopeAnalyzerErrors bool
}

func (c *container) Add(lifetime Lifetime, constructor any) Container {
	if c.err != nil {
		return c
	}

	if lifetime != Singleton &&
		lifetime != PerContext &&
		lifetime != Transient {
		c.err = LifetimeUnsupportedError(lifetime.String())

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
		r := &record{
			constructorType: withError,
			typeName:        serviceType,
			lifetime:        lifetime,
			dependencies:    constructor.Dependencies,
			constructor:     constructor.NewInstance,
		}

		c.constructorsRWM.Lock()
		defer c.constructorsRWM.Unlock()

		if _, ok := c.constructors[serviceType]; ok {
			c.err = newBadConstructorError(ErrDuplicateConstructor, t)

			return c
		}

		r.id = reflect.ValueOf(r).Pointer()
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

	cType := onlyService
	switch t.NumOut() {
	case 1:
		if out := t.Out(0); out.Implements(errorInterface) {
			c.err = newConstructorUnsupportedError(t, lifetime)

			return c
		}
	case 2:
		cType = withError

		if errType := t.Out(1); !errType.Implements(errorInterface) {
			c.err = newConstructorUnsupportedError(t, lifetime)

			return c
		}
	case 3:
		cType = withErrorAndCleanUp

		if cleanupType := t.Out(1); !cleanupType.AssignableTo(cleanUpType) {
			c.err = newConstructorUnsupportedError(t, lifetime)

			return c
		}

		if errType := t.Out(2); !errType.Implements(errorInterface) {
			c.err = newConstructorUnsupportedError(t, lifetime)

			return c
		}

		if lifetime == Transient {
			c.err = newConstructorUnsupportedError(t, lifetime)

			return c
		}
	default:
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

	r := &record{
		constructorType: cType,
		lifetime:        lifetime,
		constructor:     constructor,
		typeName:        serviceType,
	}

	for i := 0; i < numIn; i++ {
		argT := t.In(i)
		if i > 0 && argT.Implements(contextInterface) {
			c.err = newConstructorUnsupportedError(t, lifetime)

			return c
		}

		if argT.Implements(contextInterface) {
			r.dependencies = append(r.dependencies, contextDepName)
			r.dependsOnContext = true

			continue
		}

		r.dependencies = append(r.dependencies, argT.String())
	}

	r.id = reflect.ValueOf(r).Pointer()
	c.constructors[serviceType] = r

	return c
}

func (c *container) Replace(constructor any) Container {
	if c.err != nil {
		return c
	}

	var serviceType string
	if construct, ok := constructor.(func() (propertyFiller, error)); ok {
		constructor, err := construct()
		if err != nil {
			c.err = err

			return c
		}

		t := constructor.Type
		serviceType = t.String()
	} else {
		t := reflect.TypeOf(constructor)

		if t.Kind() != reflect.Func {
			c.err = newBadConstructorError(ErrConstructorNotAFunction, t)

			return c
		}

		serviceType = t.Out(0).String()

	}

	c.constructorsRWM.Lock()
	s, ok := c.constructors[serviceType]

	if !ok {
		c.err = newBadConstructorError(newConstructorNotFoundError(serviceType), reflect.TypeOf(constructor))
		c.constructorsRWM.Unlock()
		return c
	}

	delete(c.constructors, serviceType)
	c.constructorsRWM.Unlock()

	return c.Add(s.lifetime, constructor)
}

func (c *container) ServiceLocator() (ServiceLocator, error) {
	c.constructorsRWM.RLock()
	defer c.constructorsRWM.RUnlock()

	if c.err != nil {
		return nil, c.err
	}

	for _, record := range c.constructors {
		shouldBeSingleton, err := c.canResolveDependencies(*record)
		if err != nil {
			return nil, err
		}

		if !c.ignoreScopeAnalyzerErrors && shouldBeSingleton {
			logger().Error(
				"your dependency hierarchy can be optimised",
				"error", fmt.Errorf("%s %s should be a Singleton", record.lifetime, record.typeName),
			)
		}
	}

	return newLocator(c.ctx, c.constructors), nil
}

func (c *container) canResolveDependencies(record record, dependentServiceNames ...string) (bool, error) {
	dependentServiceNames = append(dependentServiceNames, record.typeName)
	shouldBeSingleton := record.lifetime < Singleton && record.dependsOnContext

	for _, dependency := range record.dependencies {
		if dependency == contextDepName {
			continue
		}

		r, ok := c.constructors[dependency]
		if !ok {
			return false, newServiceBuilderError(
				newConstructorNotFoundError(dependency),
				record.lifetime,
				record.typeName,
			)
		}

		if !c.ignoreScopeAnalyzerErrors && record.lifetime > r.lifetime {
			return false, newServiceBuilderError(
				newScopeHierarchyError(record, *r),
				record.lifetime,
				record.typeName,
			)
		}

		if shouldBeSingleton {
			shouldBeSingleton = r.lifetime == Singleton
		}

		for _, serviceName := range dependentServiceNames {
			if serviceName == dependency {
				return false, newServiceBuilderError(
					newCircularDependencyError(record.constructor, dependency),
					record.lifetime,
					record.typeName,
				)
			}
		}

		_, err := c.canResolveDependencies(*r, dependentServiceNames...)
		if err != nil {
			return false, err
		}
	}

	return shouldBeSingleton, nil
}
