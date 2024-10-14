package tinysl

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"
)

const (
	service   string = "service"
	decorator string = "decorator"
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
	id               int
	constructorType  constructorType
	lifetime         Lifetime
	dependsOnContext bool
}
type containerRecord struct {
	dependencies []string
	record
}

func newContainer(ctx context.Context, silenceUseSingletonWarnings bool) *container {
	return &container{
		ctx:                       ctx,
		constructors:              make(map[[2]string][]*containerRecord),
		ignoreScopeAnalyzerErrors: silenceUseSingletonWarnings,
		err:                       &atomic.Value{},
		nextID:                    1,
	}
}

type container struct {
	ctx                       context.Context
	err                       *atomic.Value
	constructors              map[[2]string][]*containerRecord
	constructorsRWM           sync.RWMutex
	ignoreScopeAnalyzerErrors bool
	nextID                    int
}

func (c *container) Add(lifetime Lifetime, constructor any) Container {
	if errVal := c.err.Load(); errVal != nil {
		return c
	}

	if lifetime != Singleton &&
		lifetime != PerContext &&
		lifetime != Transient {
		c.err.Store(LifetimeUnsupportedError(lifetime.String()))
		return c
	}

	// Check if constructor returns Constructor type
	construct, ok := constructor.(func() (propertyFiller, error))
	if ok {
		return c.addPropertyFiller(lifetime, service, construct)
	}

	t := reflect.TypeOf(constructor)

	cType, err := getConstructorType(lifetime, t)
	if err != nil {
		c.err.Store(err)
		return c
	}

	c.constructorsRWM.Lock()
	defer c.constructorsRWM.Unlock()

	serviceType := t.Out(0).String()
	if _, ok := c.constructors[[2]string{serviceType, service}]; ok {
		c.err.Store(newBadConstructorError(ErrDuplicateConstructor, t))
		return c
	}

	r := &containerRecord{
		record: record{
			id:              c.nextID,
			constructorType: cType,
			lifetime:        lifetime,
			constructor:     constructor,
			typeName:        serviceType,
		},
	}

	if err := fillDependencies(lifetime, t, r); err != nil {
		c.err.Store(err)
		return c
	}

	c.constructors[[2]string{serviceType, service}] = []*containerRecord{r}

	c.nextID++

	return c
}

func (c *container) Decorate(lifetime Lifetime, constructor any) Container {
	if errVal := c.err.Load(); errVal != nil {
		return c
	}

	if lifetime != Singleton &&
		lifetime != PerContext &&
		lifetime != Transient {
		c.err.Store(LifetimeUnsupportedError(lifetime.String()))
		return c
	}

	// Check if constructor returns Constructor type
	construct, ok := constructor.(func() (propertyFiller, error))
	if ok {
		return c.addPropertyFiller(lifetime, decorator, construct)
	}

	// Regular constructor
	t := reflect.TypeOf(constructor)

	cType, err := getConstructorType(lifetime, t)
	if err != nil {
		c.err.Store(err)
		return c
	}

	c.constructorsRWM.Lock()
	defer c.constructorsRWM.Unlock()

	serviceType := t.Out(0).String()
	r := &containerRecord{
		record: record{
			id:              c.nextID,
			constructorType: cType,
			lifetime:        lifetime,
			constructor:     constructor,
			typeName:        serviceType,
		},
	}

	if err := fillDependencies(lifetime, t, r); err != nil {
		c.err.Store(err)
		return c
	}

	if !slices.Contains(r.dependencies, serviceType) {
		c.err.Store(newBadConstructorError(ErrDecoratorBadDependency, t))
		return c
	}

	c.constructors[[2]string{serviceType, decorator}] = append(c.constructors[[2]string{serviceType, decorator}], r)

	c.nextID++

	return c
}

func (c *container) Replace(constructor any) Container {
	if errVal := c.err.Load(); errVal != nil {
		return c
	}

	var serviceType string
	if construct, ok := constructor.(func() (propertyFiller, error)); ok {
		constructor, err := construct()
		if err != nil {
			c.err.Store(err)

			return c
		}

		t := constructor.Type
		serviceType = t.String()
	} else {
		t := reflect.TypeOf(constructor)

		if t.Kind() != reflect.Func {
			c.err.Store(newBadConstructorError(ErrConstructorNotAFunction, t))

			return c
		}

		serviceType = t.Out(0).String()

	}

	c.constructorsRWM.Lock()
	s, ok := c.constructors[[2]string{serviceType, service}]

	if !ok || len(s) == 0 {
		c.err.Store(newBadConstructorError(newConstructorNotFoundError(serviceType), reflect.TypeOf(constructor)))
		c.constructorsRWM.Unlock()
		return c
	}

	delete(c.constructors, [2]string{serviceType, service})
	c.constructorsRWM.Unlock()

	return c.Add(s[0].lifetime, constructor)
}

func (c *container) ServiceLocator() (ServiceLocator, error) {
	c.constructorsRWM.RLock()
	defer c.constructorsRWM.RUnlock()

	if errVal := c.err.Load(); errVal != nil {
		return nil, errVal.(error)
	}

	for key, records := range c.constructors {
		for _, record := range records {
			shouldBeSingleton, err := c.canResolveDependencies(*record, key[1])
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
	}

	return newLocator(c.ctx, containerRecordsToLocatorRecords(c.constructors)), nil
}

func (c *container) canResolveDependencies(record containerRecord, role string, dependentServiceNames ...string) (bool, error) {
	dependentServiceNames = append(dependentServiceNames, record.typeName)
	shouldBeSingleton := record.lifetime < Singleton && record.dependsOnContext

	for _, dependency := range record.dependencies {
		if dependency == contextDepName {
			continue
		}

		rs, ok := c.constructors[[2]string{dependency, service}]

		switch {
		case role == decorator && dependency == record.typeName && !ok:
			return false, newServiceBuilderError(
				ErrDecoratorHasNothingToDecorate,
				record.lifetime,
				record.typeName,
			)
		case !ok:
			return false, newServiceBuilderError(
				newConstructorNotFoundError(dependency),
				record.lifetime,
				record.typeName,
			)
		}

		for _, r := range rs {
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
				if role != decorator && serviceName == dependency {
					return false, newServiceBuilderError(
						newCircularDependencyError(record.constructor, dependency),
						record.lifetime,
						record.typeName,
					)
				}
			}

			_, err := c.canResolveDependencies(*r, service, dependentServiceNames...)
			if err != nil {
				return false, err
			}
		}
	}

	return shouldBeSingleton, nil
}

func (c *container) addPropertyFiller(lifetime Lifetime, role string, construct func() (propertyFiller, error)) Container {
	constructor, err := construct()
	if err != nil {
		c.err.Store(err)

		return c
	}

	t := constructor.Type
	serviceType := t.String()
	r := &containerRecord{
		record: record{
			constructorType: withError,
			typeName:        serviceType,
			lifetime:        lifetime,
			constructor:     constructor.NewInstance,
		},
		dependencies: constructor.Dependencies,
	}

	switch role {
	case service:
		c.constructorsRWM.Lock()
		defer c.constructorsRWM.Unlock()

		if _, ok := c.constructors[[2]string{serviceType, service}]; ok {
			c.err.Store(newBadConstructorError(ErrDuplicateConstructor, t))

			return c
		}

		c.constructors[[2]string{serviceType, service}] = []*containerRecord{r}
	case decorator:
		if !slices.Contains(constructor.Dependencies, serviceType) {
			c.err.Store(newBadConstructorError(ErrDecoratorBadDependency, t))

			return c
		}

		c.constructorsRWM.Lock()
		defer c.constructorsRWM.Unlock()

		c.constructors[[2]string{serviceType, decorator}] = append(c.constructors[[2]string{serviceType, decorator}], r)
	}

	r.id = c.nextID

	c.nextID++

	return c
}

func getConstructorType(lifetime Lifetime, t reflect.Type) (constructorType, error) {
	// Regular constructor
	cType := onlyService

	if t.Kind() != reflect.Func {
		return cType, newConstructorUnsupportedError(t, lifetime)
	}

	if t.IsVariadic() {
		return cType, newBadConstructorError(ErrVariadicConstructor, t)
	}

	numIn := t.NumIn()

	// Singleton cannot be based on any context, but PerContext and Transient can
	if lifetime == Singleton && numIn > 0 && t.In(0).Implements(contextInterface) {
		return cType, newConstructorUnsupportedError(t, lifetime)
	}

	switch t.NumOut() {
	case 1:
		if out := t.Out(0); out.Implements(errorInterface) {
			return cType, newConstructorUnsupportedError(t, lifetime)
		}
	case 2:
		cType = withError

		if errType := t.Out(1); !errType.Implements(errorInterface) {
			return cType, newConstructorUnsupportedError(t, lifetime)
		}
	case 3:
		cType = withErrorAndCleanUp

		if cleanupType := t.Out(1); !cleanupType.AssignableTo(cleanUpType) {
			return cType, newConstructorUnsupportedError(t, lifetime)
		}

		if errType := t.Out(2); !errType.Implements(errorInterface) {
			return cType, newConstructorUnsupportedError(t, lifetime)
		}

		if lifetime == Transient {
			return cType, newConstructorUnsupportedError(t, lifetime)
		}
	default:
		return cType, newConstructorUnsupportedError(t, lifetime)
	}

	return cType, nil
}

func fillDependencies(lifetime Lifetime, t reflect.Type, r *containerRecord) error {
	numIn := t.NumIn()
	for i := 0; i < numIn; i++ {
		argT := t.In(i)
		if i > 0 && argT.Implements(contextInterface) {
			return newConstructorUnsupportedError(t, lifetime)
		}

		if argT.Implements(contextInterface) {
			r.dependencies = append(r.dependencies, contextDepName)
			r.dependsOnContext = true

			continue
		}

		r.dependencies = append(r.dependencies, argT.String())
	}

	return nil
}

func containerRecordsToLocatorRecords(recordsMap map[[2]string][]*containerRecord) map[string]*locatorRecord {
	result := make(map[string]*locatorRecord)

	for key, records := range recordsMap {
		if key[1] != service {
			continue
		}

		for _, value := range records {
			result[key[0]] = &locatorRecord{record: value.record}
		}
	}

	for key, records := range recordsMap {
		if key[1] != service {
			continue
		}

		for _, value := range records {
			deps := make([]*locatorRecord, len(value.dependencies))
			for i, dep := range value.dependencies {
				if dep == contextDepName {
					deps[i] = &locatorRecord{record: record{typeName: dep, id: 0}}
					continue
				}

				deps[i] = result[dep]
			}

			result[key[0]].dependencies = deps
		}
	}

	for key, records := range recordsMap {
		if key[1] != decorator {
			continue
		}

		for i, value := range records {
			deps := make([]*locatorRecord, len(value.dependencies))
			for i, dep := range value.dependencies {
				if dep == contextDepName {
					deps[i] = &locatorRecord{record: record{typeName: dep, id: 0}}
					continue
				}

				deps[i] = result[dep]
			}

			decorated := result[key[0]]
			result[key[0]] = &locatorRecord{record: value.record, dependencies: deps}
			result[fmt.Sprintf("decorated::%s::%d", key[0], i)] = decorated
		}
	}

	return result
}
