package tinysl

import (
	"context"
	"reflect"
)

var (
	// For Transient service new instance is returned.
	Transient Lifetime = "Transient"
	// For PerContext service same instance is returned for same context.Context.
	PerContext Lifetime = "PerContext"
	// For Singleton service same instance is returned always.
	Singleton Lifetime = "Singleton"
)

type Lifetime string

// ServiceLocator allows fetching service using its type name.
type ServiceLocator interface {
	// Returns service instance associated with service type name.
	Get(ctx context.Context, serviceName string) (any, error)
	// Returns error associated with ServiceLocator initialisation (if such occurred).
	Err() error
}

// Container keeps services constructors and lifetime scopes.
type Container interface {
	// Adds constructor of service with lifetime scope.
	// For Singleton constructor should be of type func(T1, T2, ...) (T, error),
	// for Transient and PerContext constructor should be of type func(context.Context, T1, T2, ...),
	// where T is exact type of service.
	Add(lifetime Lifetime, constructor any) Container
	// Returns ServiceLocator or error.
	ServiceLocator() (sl ServiceLocator, err error)
}

// Returns new Container.
func New() Container {
	return newContainer()
}

// Creates new Container, adds constructor and returns newly-created container.
func Add(lifetime Lifetime, constructor any) Container {
	return New().Add(lifetime, constructor)
}

// Returns service registered in ServiceLocator, or error if such occurred.
func Get[T any](ctx context.Context, sl ServiceLocator) (T, error) {
	var nilValue T
	serviceType := reflect.TypeOf(new(T))
	serviceName := serviceType.Elem().String()

	if err := sl.Err(); err != nil {
		return nilValue, err
	}

	s, err := sl.Get(ctx, serviceName)
	if err != nil {
		return nilValue, err
	}

	return s.(T), nil
}
