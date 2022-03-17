package tinysl_test

import (
	"context"
	"sync"

	"github.com/andriiyaremenko/tinysl"
	"github.com/stretchr/testify/suite"
)

type serviceCreationSuite struct {
	suite.Suite
}

func (suite *serviceCreationSuite) TestGetNewTransientInstance() {
	c := tinysl.New()
	i := 0
	sl, err := c.Add(tinysl.Transient, getServiceC(&i)).ServiceLocator()

	suite.NoError(err, "should not return any error")

	s1, err := tinysl.Get[service](context.TODO(), sl)

	suite.NoError(err, "should not return any error")
	suite.Equal("1 attempt", s1.Call(), "method should be invoked successfully")

	s2, err := tinysl.Get[service](context.TODO(), sl)

	suite.NoError(err, "should not return any error")
	suite.Equal("2 attempt", s2.Call(), "method should be invoked successfully")
	suite.Equal(2, i, "constructor func should have been called twice")
	suite.NotEqual(s1, s2, "transient services should not be equal")
}

func (suite *serviceCreationSuite) TestGetPerContextInstanceWithSameContext() {
	i := 0
	sl, err := tinysl.Add(tinysl.PerContext, getServiceC(&i)).ServiceLocator()

	suite.NoError(err, "should not return any error")

	ctx := context.TODO()
	s1, err := tinysl.Get[service](ctx, sl)

	suite.NoError(err, "should not return any error")
	suite.Equal("1 attempt", s1.Call(), "method should be invoked successfully")

	s2, err := tinysl.Get[service](ctx, sl)

	suite.NoError(err, "should not return any error")
	suite.Equal("1 attempt", s2.Call(), "method should be invoked successfully")
	suite.Equal(1, i, "constructor func should have been called once")
	suite.Equal(s1, s2, "PerContext services should be equal for same Context")
}

func (suite *serviceCreationSuite) TestGetPerContextInstanceWithDifferentContext() {
	i := 0
	sl, err := tinysl.Add(tinysl.PerContext, getServiceC(&i)).ServiceLocator()

	suite.NoError(err, "should not return any error")

	ctx1 := context.TODO()
	ctx2 := context.Background()

	s1, err := tinysl.Get[service](ctx1, sl)

	suite.NoError(err, "should not return any error")
	suite.Equal("1 attempt", s1.Call(), "method should be invoked successfully")

	s2, err := tinysl.Get[service](ctx2, sl)

	suite.NoError(err, "should not return any error")
	suite.Equal("2 attempt", s2.(service).Call(), "method should be invoked successfully")
	suite.Equal(2, i, "constructor func should have been called twice")
	suite.NotEqual(s1, s2, "PerContext services should not be equal for different Contexts")
}

func (suite *serviceCreationSuite) TestGetPerContextWithCancelledContext() {
	i := 0
	sl, err := tinysl.Add(tinysl.PerContext, getServiceC(&i)).ServiceLocator()

	suite.NoError(err, "should not return any error")

	ctx := context.TODO()
	ctx, cancel := context.WithCancel(ctx)
	s, err := tinysl.Get[service](ctx, sl)

	suite.NoError(err, "should not return any error")
	suite.Equal("1 attempt", s.Call(), "method should be invoked successfully")
	cancel()

	s, err = tinysl.Get[service](ctx, sl)

	suite.Nil(s, "should be nil")
	suite.Error(err, "should return an error")
	suite.Equal(1, i, "constructor func should have been called once")
}

func (suite *serviceCreationSuite) TestGetSingletonReturnsAlwaysSameInstance() {
	i := 0
	sl, err := tinysl.Add(tinysl.Singleton, getServiceC(&i)).ServiceLocator()

	suite.NoError(err, "should not return any error")

	s1, err := tinysl.Get[service](context.TODO(), sl)

	suite.NoError(err, "should not return any error")
	suite.Equal("1 attempt", s1.Call(), "method should be invoked successfully")

	s2, err := tinysl.Get[service](context.TODO(), sl)

	suite.NoError(err, "should not return any error")
	suite.Equal("1 attempt", s2.Call(), "method should be invoked successfully")
	suite.Equal(1, i, "constructor func should have been called once")
	suite.Equal(s1, s2, "singleton services should be equal")
}

func (suite *serviceCreationSuite) TestGetConstructorWithDependencies() {
	sl, err := tinysl.
		Add(tinysl.Singleton, emptyS).
		Add(tinysl.Singleton, getServiceWithDependency).
		ServiceLocator()

	s, err := tinysl.Get[service2](context.TODO(), sl)

	suite.NoError(err, "should not return any error")
	suite.NotNil(s, "should return non nil service")
}

func (suite *serviceCreationSuite) TestGetConstructorWith2Dependencies() {
	sl, err := tinysl.
		Add(tinysl.Singleton, emptyS).
		Add(tinysl.Singleton, getServiceWithDependency).
		Add(tinysl.Singleton, getServiceWith2Dependencies).
		ServiceLocator()

	suite.NoError(err, "should not return any error")

	s, err := tinysl.Get[service3](context.TODO(), sl)

	suite.NoError(err, "should not return any error")
	suite.NotNil(s, "should not return non nil service")
}

func (suite *serviceCreationSuite) TestGetConstructorWithDeepDependency() {
	sl, err := tinysl.
		Add(tinysl.Singleton, emptyS).
		Add(tinysl.Singleton, getServiceWithDependency).
		Add(tinysl.Singleton, getServiceWithDeepDependencies).
		ServiceLocator()

	suite.NoError(err, "should not return any error")

	s, err := tinysl.Get[service3](nil, sl)

	suite.NoError(err, "should not return any error")
	suite.NotNil(s, "should not return non nil service")
}

func (suite *serviceCreationSuite) TestGetConstructorWithCircularDependency() {
	_, err := tinysl.
		Add(tinysl.Singleton, getServiceWithCircularDependency).
		Add(tinysl.Singleton, getServiceWithDependency).
		Add(tinysl.Singleton, getServiceWithDeepDependencies).
		ServiceLocator()

	suite.Error(err, "should return an error")
}

func (suite *serviceCreationSuite) TestGetConstructorWithCircularDependencyInsideDependencies() {
	_, err := tinysl.
		Add(tinysl.Singleton, emptyS).
		Add(tinysl.Singleton, getServiceWithCircularDependencyInsideDependencies).
		Add(tinysl.Singleton, getServiceWithDeepDependencies).
		ServiceLocator()

	suite.Error(err, "should return an error")
}

func (suite *serviceCreationSuite) TestCanResolveDependenciesReturnsErrorOnMissingDependency() {
	_, err := tinysl.
		Add(tinysl.Singleton, emptyS).
		Add(tinysl.Singleton, getServiceWithDependency).
		Add(tinysl.Singleton, getServiceNo4).
		ServiceLocator()

	suite.Error(err, "should return an error")
}

func (suite *serviceCreationSuite) TestCanResolveDependenciesReturnsNoErrorIfOneOfDependenciesIsContext() {
	_, err := tinysl.
		Add(tinysl.PerContext, withContextC).
		Add(tinysl.PerContext, getServiceWithDependency).
		Add(tinysl.PerContext, getServiceWithDeepDependencies).
		Add(tinysl.PerContext, getServiceNo4).
		ServiceLocator()

	suite.NoError(err, "should not return any error")

}

func (suite *serviceCreationSuite) TestConcurrentGetConstructorWithDependencies() {
	c := tinysl.New()

	sl, err := c.
		Add(tinysl.PerContext, emptyS).
		Add(tinysl.PerContext, getServiceWithDependency).
		Add(tinysl.PerContext, getServiceWithDeepDependencies).
		ServiceLocator()
	suite.NoError(err, "should not return any error")

	ctx := context.TODO()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			ctx, cancel := context.WithCancel(ctx)

			defer cancel()

			s, err := tinysl.Get[service3](ctx, sl)

			suite.NoError(err, "should not return any error")
			suite.NotNil(s, "should not return non nil service")
		}()
	}

	wg.Wait()
}
