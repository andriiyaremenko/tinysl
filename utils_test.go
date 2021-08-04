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

type service3 interface {
	Call3() string
}

type service4 interface {
	Call4() string
}

type s string

func (t s) Call() string {
	return string(t)
}

func (t s) Call2() string {
	return string(t) + "_2"
}

func (t s) Call3() string {
	return string(t) + "_3"
}

func (t s) Call4() string {
	return string(t) + "_4"
}

func getServiceWithDependency(service service) (service2, error) {
	base := service.Call()
	return s(base), nil
}

func getServiceWith2Dependencies(service service, service2 service2) (service3, error) {
	begin := service.Call()
	end := service2.Call2()
	return s(begin + end), nil
}

func getServiceWithDeepDependencies(service2 service2) (service3, error) {
	base := service2.Call2()
	return s(base), nil
}

func getServiceWithCircularDependency(service3 service3) (service, error) {
	base := service3.Call3()
	return s(base), nil
}

func getServiceWithCircularDependencyInsideDependencies(service3 service3) (service2, error) {
	base := service3.Call3()
	return s(base), nil
}

func getServiceNo4(service3 service3) (service4, error) {
	base := service3.Call3()
	return s(base), nil
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

func emptyS() (service, error) {
	return s("service"), nil
}

func s2BasedOnS(sl ServiceLocator) func() (service2, error) {
	return func() (service2, error) {
		var s1 service
		_, err := sl.Get(context.TODO(), s1)

		if err != nil {
			return nil, err
		}

		return s("service_2"), nil
	}
}
