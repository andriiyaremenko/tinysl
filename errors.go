package tinysl

import (
	"context"
	"fmt"
	"reflect"

	"github.com/pkg/errors"
)

const (
	contextDepName = "context.Context"

	constructorTypeStr            string = "func(T1, T2, ...) (T, error)"
	constructorWithContextTypeStr string = "func(context.Context, T1, T2, ...) (T, error)"

	singletonPossibleConstructor  string = constructorTypeStr
	perContextPossibleConstructor string = constructorTypeStr + " | " + constructorWithContextTypeStr
	transientPossibleConstructor  string = constructorTypeStr + " | " + constructorWithContextTypeStr
)

var (
	errorInterface   = reflect.TypeOf((*error)(nil)).Elem()
	contextInterface = reflect.TypeOf((*context.Context)(nil)).Elem()

	ErrVariadicConstructor  = errors.New("variadic constructor is not supported")
	ErrDuplicateConstructor = errors.New("ServiceLocator has already registered constructor for this type")
	ErrNilContext           = errors.New("got nil context")
	ErrIWrongTType          = errors.New("I can be used only with T as a struct")
	ErrIWrongIType          = errors.New("I can be used only with I as an interface")
	ErrITDoesNotImplementI  = errors.New("I can only be used with T if T or *T implements I")
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
		return LifetimeUnsupportedError(lifetime)
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
	Lifetime                      Lifetime
	SupportedConstructorTemplates string
}

func (err *ConstructorTemplateError) Error() string {
	return fmt.Sprintf(
		"only %s can be used for %s",
		err.SupportedConstructorTemplates,
		err.Lifetime,
	)
}

func NewConstructorNotFoundError(typeName string) error {
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

func NewCircularDependencyError(constructor any, dependency string) error {
	return &CircularDependencyError{
		Dependency:  dependency,
		Constructor: constructor,
	}
}

type CircularDependencyError struct {
	Dependency  string
	Constructor any
}

func (err *CircularDependencyError) Error() string {
	return fmt.Sprintf("%s in %T is dependant on returned type", err.Dependency, err.Constructor)
}

func NewServiceBuilderError(cause error, lifetime Lifetime, typeName string) error {
	return &ServiceBuilderError{
		cause:    cause,
		Lifetime: lifetime,
		TypeName: typeName,
	}
}

type ServiceBuilderError struct {
	cause    error
	Lifetime Lifetime
	TypeName string
}

func (err *ServiceBuilderError) Error() string {
	return fmt.Sprintf("cannot build %s %s: %s", err.Lifetime, err.TypeName, err.cause)
}

func (err *ServiceBuilderError) Unwrap() error {
	return err.cause
}

func NewConstructorError(cause error) error {
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

func NewUnexpectedResultError(values []reflect.Value) error {
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
