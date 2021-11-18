package tinysl

import (
	"context"
	"sync"

	"github.com/stretchr/testify/suite"
)

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
