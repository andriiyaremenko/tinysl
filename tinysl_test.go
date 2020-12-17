package tinysl

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestServiceLocator(t *testing.T) {
	t.Run("Service can be registered as Transient", testAdd(Transient))
	t.Run("Service can be registered as PerContext", testAdd(PerContext))
	t.Run("Service can be registered as Singleton", testAdd(Singleton))
	t.Run("Service cannot be registered twice", testAddTwice)
	t.Run("Service cannot be registered twice regardless of life time", testAddTwiceDifferentLifetime)
	t.Run("Same implementation can be registered for 2 different interfaces", testSameImplementationDifferentServices)
	t.Run("Transient Service is always returned as a new instance", testTransientNewInstance)
	t.Run("PerContext Service instance is same per Context", testPerContextSameContext)
	t.Run("PerContext Service instance is different for different Contexts", testPerContextDifferentContext)
	t.Run("PerContext Service instance is not returned for cancelled Context", testPerContextCancelledContext)
	t.Run("Singleton Service instance is always the same regardless of Context", testSingletonSameInstance)
}

func testAdd(lifetime lifetime) func(*testing.T) {
	return func(t *testing.T) {
		assert := assert.New(t)

		ls := New()
		i := new(int)
		err := ls.Add(lifetime, getServiceC(i))

		assert.NoError(err, "should not return any error")
	}
}

func testAddTwice(t *testing.T) {
	assert := assert.New(t)

	ls := New()
	i := new(int)

	err := ls.Add(Transient, getServiceC(i))
	assert.NoError(err, "should not return any error")

	err = ls.Add(Transient, getServiceC(i))
	assert.Error(err, "should return an error")
}

func testAddTwiceDifferentLifetime(t *testing.T) {
	assert := assert.New(t)

	ls := New()
	i := new(int)

	err := ls.Add(Transient, getServiceC(i))
	assert.NoError(err, "should not return any error")

	err = ls.Add(PerContext, getServiceC(i))
	assert.Error(err, "should return an error")
}

func testSameImplementationDifferentServices(t *testing.T) {
	assert := assert.New(t)

	ls := New()
	i := new(int)
	ctx := context.TODO()
	var sType service
	var sType2 service2

	err := ls.Add(Transient, getServiceC(i))
	assert.NoError(err, "should not return any error")
	err = ls.Add(Transient, getServiceC2)
	assert.NoError(err, "should not return any error")

	s1, err := ls.Get(ctx, reflect.TypeOf(&sType))
	assert.NoError(err, "should not return any error")
	assert.Equal("1 attempt", s1.(service).Call(), "method should be invoked successfully")

	s2, err := ls.Get(ctx, reflect.TypeOf(&sType2))
	assert.NoError(err, "should not return any error")
	assert.Equal("service_2", s2.(service2).Call2(), "method should be invoked successfully")

	assert.NotEqual(s1, s2, "transient services should not be equal")
}

func testTransientNewInstance(t *testing.T) {
	assert := assert.New(t)

	ls := New()
	i := 0
	err := ls.Add(Transient, getServiceC(&i))
	ctx := context.TODO()
	var sType service

	s1, err := ls.Get(ctx, reflect.TypeOf(&sType))
	assert.NoError(err, "should not return any error")
	assert.Equal("1 attempt", s1.(service).Call(), "method should be invoked successfully")

	s2, err := ls.Get(ctx, reflect.TypeOf(&sType))
	assert.NoError(err, "should not return any error")
	assert.Equal("2 attempt", s2.(service).Call(), "method should be invoked successfully")

	assert.Equal(2, i, "constructor func should have been called twice")
	assert.NotEqual(s1, s2, "transient services should not be equal")
}

func testPerContextSameContext(t *testing.T) {
	assert := assert.New(t)

	ls := New()
	i := 0
	err := ls.Add(PerContext, getServiceC(&i))
	ctx := context.TODO()
	var sType service

	s1, err := ls.Get(ctx, reflect.TypeOf(&sType))
	assert.NoError(err, "should not return any error")
	assert.Equal("1 attempt", s1.(service).Call(), "method should be invoked successfully")

	s2, err := ls.Get(ctx, reflect.TypeOf(&sType))
	assert.NoError(err, "should not return any error")
	assert.Equal("1 attempt", s2.(service).Call(), "method should be invoked successfully")

	assert.Equal(1, i, "constructor func should have been called once")
	assert.Equal(s1, s2, "transient services should not be equal")
}

func testPerContextDifferentContext(t *testing.T) {
	assert := assert.New(t)

	ls := New()
	i := 0
	err := ls.Add(PerContext, getServiceC(&i))
	ctx1 := context.TODO()
	ctx2 := context.Background()
	var sType service

	s1, err := ls.Get(ctx1, reflect.TypeOf(&sType))
	assert.NoError(err, "should not return any error")
	assert.Equal("1 attempt", s1.(service).Call(), "method should be invoked successfully")

	s2, err := ls.Get(ctx2, reflect.TypeOf(&sType))
	assert.NoError(err, "should not return any error")
	assert.Equal("2 attempt", s2.(service).Call(), "method should be invoked successfully")

	assert.Equal(2, i, "constructor func should have been called twice")
	assert.NotEqual(s1, s2, "transient services should not be equal")
}

func testPerContextCancelledContext(t *testing.T) {
	assert := assert.New(t)

	ls := New()
	i := 0
	err := ls.Add(PerContext, getServiceC(&i))
	ctx := context.TODO()
	ctx, cancel := context.WithCancel(ctx)
	var sType service

	s, err := ls.Get(ctx, reflect.TypeOf(&sType))
	assert.NoError(err, "should not return any error")
	assert.Equal("1 attempt", s.(service).Call(), "method should be invoked successfully")

	cancel()

	s, err = ls.Get(ctx, reflect.TypeOf(&sType))

	assert.Nil(s, "should be nil")
	assert.Error(err, "should return an error")
	assert.Equal(1, i, "constructor func should have been called once")
}

func testSingletonSameInstance(t *testing.T) {
	assert := assert.New(t)

	ls := New()
	i := 0
	err := ls.Add(Singleton, getServiceC(&i))
	ctx1 := context.TODO()
	ctx2 := context.Background()
	var sType service

	s1, err := ls.Get(ctx1, reflect.TypeOf(&sType))
	assert.NoError(err, "should not return any error")
	assert.Equal("1 attempt", s1.(service).Call(), "method should be invoked successfully")

	s2, err := ls.Get(ctx1, reflect.TypeOf(&sType))
	assert.NoError(err, "should not return any error")
	assert.Equal("1 attempt", s2.(service).Call(), "method should be invoked successfully")

	s3, err := ls.Get(ctx2, reflect.TypeOf(&sType))
	assert.NoError(err, "should not return any error")
	assert.Equal("1 attempt", s2.(service).Call(), "method should be invoked successfully")

	assert.Equal(1, i, "constructor func should have been called once")
	assert.Equal(s1, s2, "transient services should not be equal")
	assert.Equal(s1, s3, "transient services should not be equal")
	assert.Equal(s2, s3, "transient services should not be equal")
}
