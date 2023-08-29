package tinysl

import (
	"context"
	"net/http"
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

// ServiceLocator allows fetching service using its type name.
type ServiceLocator interface {
	// Returns service instance associated with service type name.
	Get(ctx context.Context, serviceName string) (any, error)
	// Ensures ServiceLocator has service registered.
	// Will report error through ServiceLocator.Err()
	EnsureAvailable(serviceName string)
	// Reports error if ServiceLocator.EnsureAvailable(serviceName) failed to find service.
	Err() error
}

// Returns service registered in ServiceLocator, or error if such occurred.
func Get[T any](ctx context.Context, sl ServiceLocator) (T, error) {
	var nilValue T
	serviceType := reflect.TypeOf(new(T))
	serviceName := serviceType.Elem().String()

	s, err := sl.Get(ctx, serviceName)
	if err != nil {
		return nilValue, err
	}

	return s.(T), nil
}

// Returns service registered in ServiceLocator, or panics if error has occurred.
func MustGet[T any](ctx context.Context, sl ServiceLocator) T {
	s, err := Get[T](ctx, sl)
	if err != nil {
		panic(err)
	}

	return s
}

// Your HTTP handler function decorator.
// Registers an error if no constructor was found with ServiceLocator
// which should be checked with ServiceLocator.Err().
func DecorateHandler[T any](sl ServiceLocator, fn func(T) http.Handler) http.Handler {
	serviceType := reflect.TypeOf(new(T))
	serviceName := serviceType.Elem().String()

	sl.EnsureAvailable(serviceName)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		s, err := sl.Get(ctx, serviceName)
		if err != nil {
			panic(err)
		}

		fn(s.(T)).ServeHTTP(w, r)
	})
}

// Lazy initialized service.
// Will panic if error occured during initialization.
type Lazy[T any] func(context.Context) T

// Returns lazy initialization of service registered in ServiceLocator.
// Registers an error if no constructor was found with ServiceLocator
// which should be checked with ServiceLocator.Err().
func Prepare[T any](sl ServiceLocator) Lazy[T] {
	serviceType := reflect.TypeOf(new(T))
	serviceName := serviceType.Elem().String()

	sl.EnsureAvailable(serviceName)

	return func(ctx context.Context) T {
		s, err := sl.Get(ctx, serviceName)
		if err != nil {
			panic(err)
		}

		return s.(T)
	}
}
