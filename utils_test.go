package tinysl

import (
	"context"
	"fmt"
)

type service interface {
	Call() string
}

type service2 interface {
	Call2() string
}

type s string

func (t s) Call() string {
	return string(t)
}

func (t s) Call2() string {
	return string(t) + "_2"
}

func getServiceC(counter *int) func() (service, error) {
	return func() (service, error) {
		i := *counter + 1
		*counter = i
		return s(fmt.Sprintf("%d attempt", i)), nil
	}
}

func getServiceC2() (service2, error) {
	return s("service"), nil
}

func withContextC(ctx context.Context) (service, error) {
	return s("withContext"), nil
}
