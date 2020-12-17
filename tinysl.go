package tinysl

import (
	"context"
	"reflect"
)

type lifetime int

const (
	Transient lifetime = iota
	PerContext
	Singleton
)

type ServiceLocator interface {
	sealed()

	Add(lifetime, interface{}) error
	Get(context.Context, reflect.Type) (interface{}, error)
}
