package tinysl

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestServiceLocator(t *testing.T) {
	suite.Run(t, new(serviceRegistrationSuite))
	suite.Run(t, new(serviceCreationSuite))
}

type serviceRegistrationSuite struct {
	suite.Suite
}

func (suite *serviceRegistrationSuite) testAdd(lifetime lifetime) {
	ls := New()
	i := new(int)
	err := ls.Add(lifetime, getServiceC(i))

	suite.NoError(err, "should not return any error")
}

func (suite *serviceRegistrationSuite) TestAddSingleton() {
	suite.testAdd(Singleton)
}

func (suite *serviceRegistrationSuite) TestAddPerContext() {
	suite.testAdd(PerContext)
}

func (suite *serviceRegistrationSuite) TestAddTransient() {
	suite.testAdd(Transient)
}

func (suite *serviceRegistrationSuite) TestAddDuplicate() {
	ls := New()
	i := new(int)

	err := ls.Add(Transient, getServiceC(i))
	suite.NoError(err, "should not return any error")

	err = ls.Add(Transient, getServiceC(i))
	suite.Error(err, "should return an error")
}

func (suite *serviceRegistrationSuite) TestAddDuplicateDifferentLifetime() {
	ls := New()
	i := new(int)

	err := ls.Add(Transient, getServiceC(i))
	suite.NoError(err, "should not return any error")

	err = ls.Add(PerContext, getServiceC(i))
	suite.Error(err, "should return an error")
}

func (suite *serviceRegistrationSuite) TestSameImplementationDifferentInterfaces() {
	ls := New()
	i := new(int)
	var sType service
	var sType2 service2

	err := ls.Add(Transient, getServiceC(i))
	suite.NoError(err, "should not return any error")
	err = ls.Add(Transient, getServiceC2)
	suite.NoError(err, "should not return any error")

	s1, err := ls.Get(nil, &sType)
	suite.NoError(err, "should not return any error")
	suite.Equal("1 attempt", s1.(service).Call(), "method should be invoked successfully")

	s2, err := ls.Get(nil, &sType2)
	suite.NoError(err, "should not return any error")
	suite.Equal("service_2", s2.(service2).Call2(), "method should be invoked successfully")

	suite.NotEqual(s1, s2, "transient services should not be equal")
}

func (suite *serviceRegistrationSuite) TestCannotPassConstructorWithContextToSingleton() {
	ls := New()
	err := ls.Add(Singleton, withContextC)

	suite.Error(err, "should return an error")
}

func (suite *serviceRegistrationSuite) TestCanPassConstructorWithContextToPerContext() {
	ls := New()
	err := ls.Add(PerContext, withContextC)

	suite.NoError(err, "should not return an error")
}

func (suite *serviceRegistrationSuite) TestCanPassConstructorWithContextToTransient() {
	ls := New()
	err := ls.Add(Transient, withContextC)

	suite.NoError(err, "should not return an error")
}

func (suite *serviceRegistrationSuite) TestCanUseServiceLocatorInsideTransientServices() {
	sl := New()
	err := sl.Add(Transient, emptyS)
	suite.NoError(err, "should return no error")

	err = sl.Add(Transient, s2BasedOnS(sl))
	suite.NoError(err, "should return no error")
}

func (suite *serviceRegistrationSuite) TestCanUseServiceLocatorInsidePerContextServices() {
	sl := New()
	err := sl.Add(PerContext, emptyS)
	suite.NoError(err, "should return no error")

	err = sl.Add(PerContext, s2BasedOnS(sl))
	suite.NoError(err, "should return no error")
}

func (suite *serviceRegistrationSuite) TestCanUseServiceLocatorInsideSingletonServices() {
	sl := New()
	err := sl.Add(Singleton, emptyS)
	suite.NoError(err, "should return no error")

	err = sl.Add(Singleton, s2BasedOnS(sl))
	suite.NoError(err, "should return error")
}

func (suite *serviceRegistrationSuite) TestCannotAddVariadicFunctionAsConstructor() {
	sl := New()
	err := sl.Add(Singleton, func(s service, args ...string) (service2, error) {
		return getServiceC2()
	})

	suite.Error(err, "should return no error")
}

type serviceCreationSuite struct {
	suite.Suite
}

func (suite *serviceCreationSuite) TestGetNewTransientInstance() {
	ls := New()
	i := 0
	err := ls.Add(Transient, getServiceC(&i))

	suite.NoError(err, "should not return any error")

	var sType service
	s1, err := ls.Get(nil, &sType)

	suite.NoError(err, "should not return any error")
	suite.Equal("1 attempt", s1.(service).Call(), "method should be invoked successfully")

	s2, err := ls.Get(nil, &sType)

	suite.NoError(err, "should not return any error")
	suite.Equal("2 attempt", s2.(service).Call(), "method should be invoked successfully")
	suite.Equal(2, i, "constructor func should have been called twice")
	suite.NotEqual(s1, s2, "transient services should not be equal")
}

func (suite *serviceCreationSuite) TestGetPerContextInstanceWithSameContext() {
	ls := New()
	i := 0
	err := ls.Add(PerContext, getServiceC(&i))

	suite.NoError(err, "should not return any error")

	ctx := context.TODO()
	var sType service

	s1, err := ls.Get(ctx, &sType)

	suite.NoError(err, "should not return any error")
	suite.Equal("1 attempt", s1.(service).Call(), "method should be invoked successfully")

	s2, err := ls.Get(ctx, &sType)

	suite.NoError(err, "should not return any error")
	suite.Equal("1 attempt", s2.(service).Call(), "method should be invoked successfully")
	suite.Equal(1, i, "constructor func should have been called once")
	suite.Equal(s1, s2, "PerContext services should be equal for same Context")
}

func (suite *serviceCreationSuite) TestGetPerContextInstanceWithDifferentContext() {
	ls := New()
	i := 0
	err := ls.Add(PerContext, getServiceC(&i))

	suite.NoError(err, "should not return any error")

	ctx1 := context.TODO()
	ctx2 := context.Background()

	var sType service
	s1, err := ls.Get(ctx1, &sType)

	suite.NoError(err, "should not return any error")
	suite.Equal("1 attempt", s1.(service).Call(), "method should be invoked successfully")

	s2, err := ls.Get(ctx2, &sType)

	suite.NoError(err, "should not return any error")
	suite.Equal("2 attempt", s2.(service).Call(), "method should be invoked successfully")
	suite.Equal(2, i, "constructor func should have been called twice")
	suite.NotEqual(s1, s2, "PerContext services should not be equal for different Contexts")
}

func (suite *serviceCreationSuite) TestGetPerContextWithCancelledContext() {
	ls := New()
	i := 0
	err := ls.Add(PerContext, getServiceC(&i))

	suite.NoError(err, "should not return any error")

	var sType service
	ctx := context.TODO()
	ctx, cancel := context.WithCancel(ctx)
	s, err := ls.Get(ctx, &sType)

	suite.NoError(err, "should not return any error")
	suite.Equal("1 attempt", s.(service).Call(), "method should be invoked successfully")
	cancel()

	s, err = ls.Get(ctx, &sType)

	suite.Nil(s, "should be nil")
	suite.Error(err, "should return an error")
	suite.Equal(1, i, "constructor func should have been called once")
}

func (suite *serviceCreationSuite) TestGetSingletonReturnsAlwaysSameInstance() {
	ls := New()
	i := 0
	err := ls.Add(Singleton, getServiceC(&i))

	suite.NoError(err, "should not return any error")

	var sType service
	s1, err := ls.Get(nil, &sType)

	suite.NoError(err, "should not return any error")
	suite.Equal("1 attempt", s1.(service).Call(), "method should be invoked successfully")

	s2, err := ls.Get(nil, &sType)

	suite.NoError(err, "should not return any error")
	suite.Equal("1 attempt", s2.(service).Call(), "method should be invoked successfully")
	suite.Equal(1, i, "constructor func should have been called once")
	suite.Equal(s1, s2, "singleton services should be equal")
}

func (suite *serviceCreationSuite) TestGetConstructorWithDependencies() {
	ls := New()

	err := ls.Add(Singleton, emptyS)
	suite.NoError(err, "should not return any error")

	err = ls.Add(Singleton, getServiceWithDependency)
	suite.NoError(err, "should not return any error")

	var sType service2
	s, err := ls.Get(nil, &sType)

	suite.NoError(err, "should not return any error")
	suite.NotNil(s, "should return non nil service")
}

func (suite *serviceCreationSuite) TestGetConstructorWith2Dependencies() {
	ls := New()

	err := ls.Add(Singleton, emptyS)
	suite.NoError(err, "should not return any error")

	err = ls.Add(Singleton, getServiceWithDependency)
	suite.NoError(err, "should not return any error")

	err = ls.Add(Singleton, getServiceWith2Dependencies)
	suite.NoError(err, "should not return any error")

	var sType service3
	s, err := ls.Get(nil, &sType)

	suite.NoError(err, "should not return any error")
	suite.NotNil(s, "should not return non nil service")
}

func (suite *serviceCreationSuite) TestGetConstructorWithDeepDependency() {
	ls := New()

	err := ls.Add(Singleton, emptyS)
	suite.NoError(err, "should not return any error")

	err = ls.Add(Singleton, getServiceWithDependency)
	suite.NoError(err, "should not return any error")

	err = ls.Add(Singleton, getServiceWithDeepDependencies)
	suite.NoError(err, "should not return any error")

	var sType service3
	s, err := ls.Get(nil, &sType)

	suite.NoError(err, "should not return any error")
	suite.NotNil(s, "should not return non nil service")
}

func (suite *serviceCreationSuite) TestGetConstructorWithCircularDependency() {
	ls := New()

	err := ls.Add(Singleton, getServiceWithCircularDependency)
	suite.NoError(err, "should not return any error")

	err = ls.Add(Singleton, getServiceWithDependency)
	suite.NoError(err, "should not return any error")

	err = ls.Add(Singleton, getServiceWithDeepDependencies)
	suite.NoError(err, "should not return any error")

	var sType service3
	s, err := ls.Get(nil, &sType)

	suite.Error(err, "should return an error")
	suite.Nil(s, "should return nil")
}

func (suite *serviceCreationSuite) TestGetConstructorWithCircularDependencyInsideDependencies() {
	ls := New()

	err := ls.Add(Singleton, emptyS)
	suite.NoError(err, "should not return any error")

	err = ls.Add(Singleton, getServiceWithCircularDependencyInsideDependencies)
	suite.NoError(err, "should not return any error")

	err = ls.Add(Singleton, getServiceWithDeepDependencies)
	suite.NoError(err, "should not return any error")

	var sType service3
	s, err := ls.Get(nil, &sType)

	suite.Error(err, "should return an error")
	suite.Nil(s, "should return nil")
}

func (suite *serviceCreationSuite) TestCanResolveDependenciesReturnsErrorOnMissingDependency() {
	ls := New()

	err := ls.Add(Singleton, emptyS)
	suite.NoError(err, "should not return any error")

	err = ls.Add(Singleton, getServiceWithDependency)
	suite.NoError(err, "should not return any error")

	err = ls.Add(Singleton, getServiceNo4)
	suite.NoError(err, "should not return any error")

	err = ls.CanResolveDependencies()
	suite.Error(err, "should return an error")
}

func (suite *serviceCreationSuite) TestCanResolveDependenciesReturnsNoErrorIfAllDependenciesPresent() {
	ls := New()

	err := ls.Add(Singleton, emptyS)
	suite.NoError(err, "should not return any error")

	err = ls.Add(Singleton, getServiceWithDependency)
	suite.NoError(err, "should not return any error")

	err = ls.Add(Singleton, getServiceWithDeepDependencies)
	suite.NoError(err, "should not return any error")

	err = ls.Add(Singleton, getServiceNo4)
	suite.NoError(err, "should not return any error")

	err = ls.CanResolveDependencies()
	suite.NoError(err, "should not return an error")
}

func (suite *serviceCreationSuite) TestCanResolveDependenciesReturnsNoErrorIfOneOfDependenciesIsContext() {
	ls := New()

	err := ls.Add(PerContext, withContextC)
	suite.NoError(err, "should not return any error")

	err = ls.Add(PerContext, getServiceWithDependency)
	suite.NoError(err, "should not return any error")

	err = ls.Add(PerContext, getServiceWithDeepDependencies)
	suite.NoError(err, "should not return any error")

	err = ls.Add(PerContext, getServiceNo4)
	suite.NoError(err, "should not return any error")

	err = ls.CanResolveDependencies()
	suite.NoError(err, "should not return an error")
}

func (suite *serviceCreationSuite) TestCanResolveDependenciesReturnsErrorIfThereIsCircularDependency() {
	ls := New()

	err := ls.Add(Singleton, getServiceWithCircularDependency)
	suite.NoError(err, "should not return any error")

	err = ls.Add(Singleton, getServiceWithDependency)
	suite.NoError(err, "should not return any error")

	err = ls.Add(Singleton, getServiceWithDeepDependencies)
	suite.NoError(err, "should not return any error")

	err = ls.Add(Singleton, getServiceNo4)
	suite.NoError(err, "should not return any error")

	err = ls.CanResolveDependencies()
	suite.Error(err, "should return an error")
}
