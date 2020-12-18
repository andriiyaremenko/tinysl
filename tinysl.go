package tinysl

// This package provides simple abstraction to manage lifetime scope of services.
// This package does NOT try to be another IOC container.
// It was created because of need to share same instances of services among gorutines
// within lifetime of a context.
// `PerContext` lifetime scope was main reason to create this package,
// other scopes were created for convenience.

import (
	"context"
)

type lifetime int

const (
	// For `Transient` service new instance is returned.
	Transient lifetime = iota
	// For `PerContext` service same instance is returned for same `context.Context`.
	PerContext
	// For `Singleton` service same instance is returned always.
	Singleton
)

// Helps manage lifetime scope of services.
// This interface is sealed.
type ServiceLocator interface {
	sealed()
	// Adds constructor of service with `lifetime` scope.
	// For `Transient` and `Singleton` `constructor` should be of type `func() (T, error)`,
	// for `PerContext` `constructor` should be of type `func ()|(context.Context) (T, error)`,
	// where `T` is exact type of service.
	Add(lifetime lifetime, constructor interface{}) error
	// Returns service in from of `interface{}`.
	// Should be upcased to service type to use.
	// `ctx` is used for `PerContext` scoped services in other cases can be `nil`.
	// `servicePrt` should be pointer to a `T`, where `T` is type of service to be constructed.
	Get(ctx context.Context, servicePtr interface{}) (service interface{}, err error)
}
