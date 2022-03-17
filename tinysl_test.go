package tinysl_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestServiceLocator(t *testing.T) {
	suite.Run(t, new(serviceRegistrationSuite))
	suite.Run(t, new(serviceCreationSuite))
}
