package tinysl

import (
	"context"
	"reflect"
)

const (
	contextDepName = "context.Context"

	containerIsNotInstantiated     string = "Container is not instantiated"
	unsupportedLifetime            string = "%s Lifetime is unsupported"
	wrongConstructor               string = "constructor should be of type %s for %s, got %s"
	constructorNotFound            string = "constructor for %s not found"
	constructorReturnedError       string = "constructor %T returned an error"
	constructorReturnedBadResult   string = "constructor %T returned an unexpected result: %v"
	variadicConstructorUnsupported string = "variadic function as constructor is not supported: got %s"
	duplicateConstructor           string = "ServiceLocator has already registered constructor for %s: %T"
	notAPointer                    string = "service type should be pointer type, got: %s"
	missingDependency              string = "%s has unregistered dependency: %s"
	circularDependencyFound        string = "circular dependency in %T: %s depends on %s"
	nilContext                     string = "PerContext service %s cannot be served for nil context"
	cancelledContext               string = "PerContext service %s cannot be served for cancelled context"

	constructorTypeStr            string = "func(T1, T2, ...) (T, error)"
	constructorWithContextTypeStr string = "func(context.Context, T1, T2, ...) (T, error)"

	singletonPossibleConstructor  string = constructorTypeStr
	perContextPossibleConstructor string = constructorTypeStr + " | " + constructorWithContextTypeStr
	transientPossibleConstructor  string = constructorTypeStr + " | " + constructorWithContextTypeStr
)

var errorInterface = reflect.TypeOf((*error)(nil)).Elem()
var contextInterface = reflect.TypeOf((*context.Context)(nil)).Elem()
