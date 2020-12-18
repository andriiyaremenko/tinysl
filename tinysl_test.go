package tinysl

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceLocator(t *testing.T) {
	t.Run("Service can be registered as Transient", testAdd(Transient))
	t.Run("Service can be registered as PerContext", testAdd(PerContext))
	t.Run("Service can be registered as Singleton", testAdd(Singleton))
	t.Run("Service cannot be registered twice", testAddTwice)
	t.Run("Service cannot be registered twice regardless of life time", testAddTwiceDifferentLifetime)
	t.Run("Same implementation can be registered for 2 different interfaces",
		testSameImplementationDifferentServices)
	t.Run("Transient Service is always returned as a new instance", testTransientNewInstance)
	t.Run("PerContext Service instance is same per Context", testPerContextSameContext)
	t.Run("PerContext Service instance is different for different Contexts",
		testPerContextDifferentContext)
	t.Run("PerContext Service instance is not returned for cancelled Context",
		testPerContextCancelledContext)
	t.Run("Singleton Service instance is always the same regardless of Context",
		testSingletonSameInstance)
	t.Run("Can pass constructor with Context argument", testPassConstructorWithContext)
	t.Run("Cannot pass constructor with more than 1 argument or argument not of type Context",
		testCanPassNoArgsOrWithContextConstructor)
	t.Run("Can pass constructor with Context argument only with PerContext lifetime",
		testPassConstructorWithContextOnlyToPerContext)
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
	var sType service
	var sType2 service2

	err := ls.Add(Transient, getServiceC(i))
	assert.NoError(err, "should not return any error")
	err = ls.Add(Transient, getServiceC2)
	assert.NoError(err, "should not return any error")

	s1, err := ls.Get(nil, &sType)
	assert.NoError(err, "should not return any error")
	assert.Equal("1 attempt", s1.(service).Call(), "method should be invoked successfully")

	s2, err := ls.Get(nil, &sType2)
	assert.NoError(err, "should not return any error")
	assert.Equal("service_2", s2.(service2).Call2(), "method should be invoked successfully")

	assert.NotEqual(s1, s2, "transient services should not be equal")
}

func testTransientNewInstance(t *testing.T) {
	assert := assert.New(t)

	ls := New()
	i := 0
	err := ls.Add(Transient, getServiceC(&i))
	assert.NoError(err, "should not return any error")
	var sType service

	s1, err := ls.Get(nil, &sType)
	assert.NoError(err, "should not return any error")
	assert.Equal("1 attempt", s1.(service).Call(), "method should be invoked successfully")

	s2, err := ls.Get(nil, &sType)
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
	assert.NoError(err, "should not return any error")
	ctx := context.TODO()
	var sType service

	s1, err := ls.Get(ctx, &sType)
	assert.NoError(err, "should not return any error")
	assert.Equal("1 attempt", s1.(service).Call(), "method should be invoked successfully")

	s2, err := ls.Get(ctx, &sType)
	assert.NoError(err, "should not return any error")
	assert.Equal("1 attempt", s2.(service).Call(), "method should be invoked successfully")

	assert.Equal(1, i, "constructor func should have been called once")
	assert.Equal(s1, s2, "PerContext services should be equal for same Context")
}

func testPerContextDifferentContext(t *testing.T) {
	assert := assert.New(t)

	ls := New()
	i := 0
	err := ls.Add(PerContext, getServiceC(&i))
	assert.NoError(err, "should not return any error")
	ctx1 := context.TODO()
	ctx2 := context.Background()
	var sType service

	s1, err := ls.Get(ctx1, &sType)
	assert.NoError(err, "should not return any error")
	assert.Equal("1 attempt", s1.(service).Call(), "method should be invoked successfully")

	s2, err := ls.Get(ctx2, &sType)
	assert.NoError(err, "should not return any error")
	assert.Equal("2 attempt", s2.(service).Call(), "method should be invoked successfully")

	assert.Equal(2, i, "constructor func should have been called twice")
	assert.NotEqual(s1, s2, "PerContext services should not be equal for different Contexts")
}

func testPerContextCancelledContext(t *testing.T) {
	assert := assert.New(t)

	ls := New()
	i := 0
	err := ls.Add(PerContext, getServiceC(&i))
	assert.NoError(err, "should not return any error")
	ctx := context.TODO()
	ctx, cancel := context.WithCancel(ctx)
	var sType service

	s, err := ls.Get(ctx, &sType)
	assert.NoError(err, "should not return any error")
	assert.Equal("1 attempt", s.(service).Call(), "method should be invoked successfully")

	cancel()

	s, err = ls.Get(ctx, &sType)

	assert.Nil(s, "should be nil")
	assert.Error(err, "should return an error")
	assert.Equal(1, i, "constructor func should have been called once")
}

func testSingletonSameInstance(t *testing.T) {
	assert := assert.New(t)

	ls := New()
	i := 0
	err := ls.Add(Singleton, getServiceC(&i))
	assert.NoError(err, "should not return any error")
	var sType service

	s1, err := ls.Get(nil, &sType)
	assert.NoError(err, "should not return any error")
	assert.Equal("1 attempt", s1.(service).Call(), "method should be invoked successfully")

	s2, err := ls.Get(nil, &sType)
	assert.NoError(err, "should not return any error")
	assert.Equal("1 attempt", s2.(service).Call(), "method should be invoked successfully")

	assert.Equal(1, i, "constructor func should have been called once")
	assert.Equal(s1, s2, "singleton services should be equal")
}

func testPassConstructorWithContext(t *testing.T) {
	assert := assert.New(t)

	ls := New()
	var sType service

	err := ls.Add(PerContext, withContextC)
	assert.NoError(err, "should not return any error")

	s, err := ls.Get(context.TODO(), &sType)
	assert.NoError(err, "should not return any error")
	assert.Equal("withContext", s.(service).Call(), "method should be invoked successfully")
}

func testCanPassNoArgsOrWithContextConstructor(t *testing.T) {
	assert := assert.New(t)
	badConstructor1 := func(ctx context.Context, i int) (service, error) {
		return withContextC(ctx)
	}

	ls := New()
	err := ls.Add(PerContext, badConstructor1)
	assert.Error(err, "should return an error")
	err = ls.Add(PerContext, getServiceC)
	assert.Error(err, "should return an error")
}

func testPassConstructorWithContextOnlyToPerContext(t *testing.T) {
	assert := assert.New(t)

	ls := New()
	err := ls.Add(Transient, withContextC)
	assert.Error(err, "should return an error")
	err = ls.Add(Singleton, withContextC)
	assert.Error(err, "should return an error")
}
