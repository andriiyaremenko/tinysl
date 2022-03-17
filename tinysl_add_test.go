package tinysl_test

import (
	"context"
	"sync"

	"github.com/andriiyaremenko/tinysl"
	"github.com/stretchr/testify/suite"
)

type serviceRegistrationSuite struct {
	suite.Suite
}

func (suite *serviceRegistrationSuite) TestAddSingleton() {
	i := new(int)
	_, err := tinysl.Add(tinysl.Singleton, getServiceC(i)).ServiceLocator()

	suite.NoError(err, "should not return any error")
}

func (suite *serviceRegistrationSuite) TestAddPerContext() {
	i := new(int)
	_, err := tinysl.Add(tinysl.PerContext, getServiceC(i)).ServiceLocator()

	suite.NoError(err, "should not return any error")
}

func (suite *serviceRegistrationSuite) TestAddTransient() {
	i := new(int)
	_, err := tinysl.Add(tinysl.Transient, getServiceC(i)).ServiceLocator()

	suite.NoError(err, "should not return any error")
}

func (suite *serviceRegistrationSuite) TestAddDuplicate() {
	i := new(int)
	_, err := tinysl.
		Add(tinysl.Transient, getServiceC(i)).
		Add(tinysl.Transient, getServiceC(i)).
		ServiceLocator()

	suite.Error(err, "should return an error")
}

func (suite *serviceRegistrationSuite) TestAddDuplicateDifferentLifetime() {
	i := new(int)

	_, err := tinysl.
		Add(tinysl.Transient, getServiceC(i)).
		Add(tinysl.PerContext, getServiceC(i)).
		ServiceLocator()

	suite.Error(err, "should return an error")
}

func (suite *serviceRegistrationSuite) TestSameImplementationDifferentInterfaces() {
	i := new(int)

	c, err := tinysl.
		Add(tinysl.Transient, getServiceC(i)).
		Add(tinysl.Transient, getServiceC2).
		ServiceLocator()
	suite.NoError(err, "should not return any error")

	s1, err := tinysl.Get[service](context.TODO(), c)

	suite.NoError(err, "should not return any error")
	suite.Equal("1 attempt", s1.Call(), "method should be invoked successfully")

	s2, err := tinysl.Get[service2](context.TODO(), c)

	suite.NoError(err, "should not return any error")
	suite.Equal("service_2", s2.(service2).Call2(), "method should be invoked successfully")
	suite.NotEqual(s1, s2, "transient services should not be equal")
}

func (suite *serviceRegistrationSuite) TestCannotPassConstructorWithContextToSingleton() {
	_, err := tinysl.Add(tinysl.Singleton, withContextC).ServiceLocator()

	suite.Error(err, "should return an error")
}

func (suite *serviceRegistrationSuite) TestCanPassConstructorWithContextToPerContext() {
	_, err := tinysl.Add(tinysl.PerContext, withContextC).ServiceLocator()

	suite.NoError(err, "should not return an error")
}

func (suite *serviceRegistrationSuite) TestCanPassConstructorWithContextToTransient() {
	_, err := tinysl.Add(tinysl.Transient, withContextC).ServiceLocator()

	suite.NoError(err, "should not return an error")
}

func (suite *serviceRegistrationSuite) TestCannotAddVariadicFunctionAsConstructor() {
	_, err := tinysl.
		Add(tinysl.Singleton, func(s service, args ...string) (service2, error) {
			return getServiceC2()
		}).
		ServiceLocator()

	suite.Error(err, "should return no error")
}

func (suite *serviceRegistrationSuite) TestAddConcurrently() {
	sl := tinysl.New()

	var wg sync.WaitGroup

	var err1 error
	wg.Add(1)
	go func() {
		_, err1 = sl.Add(tinysl.Transient, getServiceC2).ServiceLocator()

		wg.Done()
	}()

	var err2 error
	wg.Add(1)
	go func() {
		_, err2 = sl.Add(tinysl.Transient, getServiceC2).ServiceLocator()

		wg.Done()
	}()

	wg.Wait()

	if err1 == nil && err2 == nil {
		suite.Fail("duplicate was registered in concurrent write")
	}
}
