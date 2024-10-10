package tinysl

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"runtime/debug"
	"sync/atomic"
)

const (
	// For Transient service new instance is returned.
	Transient Lifetime = iota
	// For PerContext service same instance is returned for same context.Context.
	PerContext
	// For Singleton service same instance is returned always.
	Singleton
)

func (l Lifetime) String() string {
	switch l {
	case Transient:
		return "Transient"
	case PerContext:
		return "PerContext"
	case Singleton:
		return "Singleton"
	default:
		return "Unsupported"
	}
}

type Lifetime int

type Cleanup func()

func (c Cleanup) CallWithRecovery(l Lifetime) {
	defer func() {
		if rp := recover(); rp != nil {
			logger().Error(
				fmt.Sprintf("recovered from panic during %s cleanup", l),
				"error", newRecoveredError(rp, debug.Stack()))
		}
	}()

	c()
}

type Logger interface {
	Error(msg string, args ...any)
}

func init() {
	loggerPtr.Store(func() *Logger { var l Logger = slog.Default(); return &l }())
}

var (
	goHasMovingGC    atomic.Bool
	enableStackTrace atomic.Bool
	loggerPtr        atomic.Pointer[Logger]
)

func SetGoHasMovingGC() {
	goHasMovingGC.Store(true)
}

func EnableStackTrace() {
	enableStackTrace.Store(true)
}

func SetLogger(l Logger) {
	if l != nil {
		loggerPtr.Store(&l)
	}
}

func logger() Logger {
	return *loggerPtr.Load()
}

// Container keeps services constructors and lifetime scopes.
type Container interface {
	// Adds constructor of service with lifetime scope.
	// For Singleton constructor should be of type func(T1, T2, ...) (T, error),
	// for Transient and PerContext constructor should be of type func(context.Context, T1, T2, ...),
	// where T is exact type of service.
	Add(lifetime Lifetime, constructor any) Container
	// Decorate constructor of service.
	Decorate(lifetime Lifetime, constructor any) Container
	// Replaces constructor of service with same lifetime as registered before.
	Replace(constructor any) Container
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

// Your HTTP middleware function decorator.
// Registers an error if no constructor was found with ServiceLocator
// which should be checked with ServiceLocator.Err().
func DecorateMiddleware[T any](sl ServiceLocator, fn func(T) func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	serviceType := reflect.TypeOf(new(T))
	serviceName := serviceType.Elem().String()

	sl.EnsureAvailable(serviceName)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			s, err := sl.Get(ctx, serviceName)
			if err != nil {
				panic(err)
			}

			fn(s.(T))(next).ServeHTTP(w, r)
		})
	}
}

// Your HTTP handler function decorator.
// Registers an error if no constructor was found with ServiceLocator
// which should be checked with ServiceLocator.Err().
func DecorateHandler[T any, H http.Handler](sl ServiceLocator, fn func(T) H) http.HandlerFunc {
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
// Will panic if error occurred during initialization.
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
