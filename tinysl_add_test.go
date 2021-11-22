package tinysl

import "github.com/stretchr/testify/suite"

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

