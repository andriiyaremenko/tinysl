package tinysl

import (
	"context"
	"fmt"
	"reflect"
)

const (
	contextDepName = "context.Context"

	constructorTypeStr            string = "func(T1, ...) [T|(T, error)|(T, func(), error)]"
	constructorWithContextTypeStr string = "func(context.Context, T1, ...) [T|(T, error)|(T, func(), error)]"

	singletonPossibleConstructor  string = constructorTypeStr
	perContextPossibleConstructor string = constructorTypeStr + " | " + constructorWithContextTypeStr
	transientPossibleConstructor  string = "func(T1, ...) [T|(T, error)]" + " | " + "func(context.Context, T1, ...) [T|(T, error)]"
)

var (
	errorInterface   = reflect.TypeOf((*error)(nil)).Elem()
	cleanUpType      = reflect.TypeOf((*func())(nil)).Elem()
	contextInterface = reflect.TypeOf((*context.Context)(nil)).Elem()

	ErrVariadicConstructor  = fmt.Errorf("variadic constructor is not supported")
	ErrDuplicateConstructor = fmt.Errorf("ServiceLocator has already registered constructor for this type")
	ErrNilContext           = fmt.Errorf("got nil context")
	ErrIWrongTType          = fmt.Errorf("I can be used only with T as a struct")
	ErrIWrongIType          = fmt.Errorf("I can be used only with I as an interface")
	ErrITDoesNotImplementI  = fmt.Errorf("I can only be used with T if T or *T implements I")
)

func newConstructorUnsupportedError(constructorType reflect.Type, lifetime Lifetime) error {
	switch lifetime {
	case Singleton:
		return newBadConstructorError(
			&ConstructorTemplateError{
				Lifetime:                      lifetime,
				SupportedConstructorTemplates: singletonPossibleConstructor,
			},
			constructorType,
		)
	case PerContext:
		return newBadConstructorError(
			&ConstructorTemplateError{
				Lifetime:                      lifetime,
				SupportedConstructorTemplates: perContextPossibleConstructor,
			},
			constructorType,
		)
	case Transient:
		return newBadConstructorError(
			&ConstructorTemplateError{
				Lifetime:                      lifetime,
				SupportedConstructorTemplates: transientPossibleConstructor,
			},
			constructorType,
		)
	default:
		return LifetimeUnsupportedError(lifetime.String())
	}
}

type LifetimeUnsupportedError string

func (lifetime LifetimeUnsupportedError) Error() string {
	return fmt.Sprintf("%s Lifetime is unsupported", string(lifetime))
}

func newBadConstructorError(cause error, constructorType reflect.Type) error {
	return &BadConstructorError{
		cause:           cause,
		ConstructorType: constructorType,
	}
}

type BadConstructorError struct {
	cause           error
	ConstructorType reflect.Type
}

func (err *BadConstructorError) Error() string {
	return fmt.Sprintf("bad constructor %s: %s", err.ConstructorType, err.cause)
}

func (err *BadConstructorError) Unwrap() error {
	return err.cause
}

type TError struct {
	T reflect.Type
}

func (err *TError) Error() string {
	return fmt.Sprintf("tinysl.T can only be used with a struct, got %s", err.T)
}

type PError struct {
	T reflect.Type
}

func (err *PError) Error() string {
	return fmt.Sprintf("tinysl.P can only be used with a struct, got %s", err.T)
}

func newIError(cause error, i, t reflect.Type) error {
	return &IError{T: t, I: i, cause: cause}
}

type IError struct {
	cause error

	I, T reflect.Type
}

func (err *IError) Error() string {
	return fmt.Sprintf("tinysl.I[%s, %s] returned an error: %s", err.I, err.T, err.cause)
}

func (err *IError) Unwrap() error {
	return err.cause
}

type ConstructorTemplateError struct {
	SupportedConstructorTemplates string
	Lifetime                      Lifetime
}

func (err *ConstructorTemplateError) Error() string {
	return fmt.Sprintf(
		"only %s can be used for %s",
		err.SupportedConstructorTemplates,
		err.Lifetime,
	)
}

func newConstructorNotFoundError(typeName string) error {
	return &ConstructorNotFoundError{
		TypeName: typeName,
	}
}

type ConstructorNotFoundError struct {
	TypeName string
}

func (err *ConstructorNotFoundError) Error() string {
	return fmt.Sprintf("%s constructor not found", err.TypeName)
}

func newCircularDependencyError(constructor any, dependency string) error {
	return &CircularDependencyError{
		Dependency:  dependency,
		Constructor: constructor,
	}
}

type CircularDependencyError struct {
	Constructor any
	Dependency  string
}

func (err *CircularDependencyError) Error() string {
	return fmt.Sprintf("%s in %T is dependant on returned type", err.Dependency, err.Constructor)
}

func newScopeHierarchyError(rec, dep record) error {
	return &ScopeHierarchyError{DepServiceName: rec.typeName, DepLifetime: dep.lifetime}
}

type ScopeHierarchyError struct {
	DepServiceName string
	DepLifetime    Lifetime
}

func (err *ScopeHierarchyError) Error() string {
	return fmt.Sprintf(
		"dependency on %s %s violates scope hierarchy",
		err.DepServiceName,
		err.DepLifetime,
	)
}

func newServiceBuilderError(cause error, lifetime Lifetime, typeName string) error {
	return &ServiceBuilderError{
		cause:    cause,
		Lifetime: lifetime,
		TypeName: typeName,
	}
}

type ServiceBuilderError struct {
	cause    error
	TypeName string
	Lifetime Lifetime
}

func (err *ServiceBuilderError) Error() string {
	return fmt.Sprintf("cannot build %s %s: %s", err.Lifetime, err.TypeName, err.cause)
}

func (err *ServiceBuilderError) Unwrap() error {
	return err.cause
}

func newConstructorError(cause error) error {
	return &ConstructorError{
		cause: cause,
	}
}

type ConstructorError struct {
	cause error
}

func (err *ConstructorError) Error() string {
	return fmt.Sprintf("constructor returned an error: %s", err.cause)
}

func (err *ConstructorError) Unwrap() error {
	return err.cause
}

func newUnexpectedResultError(values []reflect.Value) error {
	return &UnexpectedResultError{
		Result: values,
	}
}

type UnexpectedResultError struct {
	Result []reflect.Value
}

func (err *UnexpectedResultError) Error() string {
	return fmt.Sprintf("unexpected result: %#v", err.Result)
}
