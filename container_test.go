package tinysl_test

import (
	"context"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/andriiyaremenko/tinysl"
)

var _ = Describe("Container", func() {
	It("should register Singleton", func() {
		_, err := tinysl.Add(tinysl.Singleton, nameProviderConstructor).ServiceLocator()
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("should refuse register Singleton constructor dependant on context.Context", func() {
		_, err := tinysl.
			Add(tinysl.Singleton, nameServiceConstructor).
			Add(tinysl.Singleton, tableTimerConstructor).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.BadConstructorError)))
		Expect(errors.Unwrap(err)).Should(BeAssignableToTypeOf(new(tinysl.ConstructorTemplateError)))
	})

	It("should register PerContext", func() {
		_, err := tinysl.Add(tinysl.PerContext, nameProviderConstructor).ServiceLocator()
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("should register PerContext constructor dependant on context.Context", func() {
		_, err := tinysl.
			Add(tinysl.PerContext, tableTimerConstructor).
			Add(tinysl.Singleton, nameServiceConstructor).
			ServiceLocator()
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("should register Transient", func() {
		_, err := tinysl.Add(tinysl.Transient, nameProviderConstructor).ServiceLocator()
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("should register Transient constructor dependant on context.Context", func() {
		_, err := tinysl.
			Add(tinysl.Transient, tableTimerConstructor).
			Add(tinysl.Singleton, nameServiceConstructor).
			ServiceLocator()
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("should not allow add duplicate services for same lifetime", func() {
		_, err := tinysl.
			Add(tinysl.Transient, nameProviderConstructor).
			Add(tinysl.Transient, nameProviderConstructor).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.BadConstructorError)))
		Expect(errors.Unwrap(err)).Should(MatchError(tinysl.ErrDuplicateConstructor))
	})

	It("should allow to use same implementation for different types", func() {
		_, err := tinysl.
			Add(tinysl.Transient, nameProviderConstructor).
			Add(tinysl.PerContext, nameServiceConstructor).
			ServiceLocator()
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("should refuse register variadic constructors", func() {
		variadicConstructor := func(args ...any) (nameService, error) {
			return nameProvider("Bob"), nil
		}
		_, err := tinysl.
			Add(tinysl.Transient, variadicConstructor).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.BadConstructorError)))
		Expect(errors.Unwrap(err)).Should(MatchError(tinysl.ErrVariadicConstructor))
	})

	It("should be tread-safe", func() {
		sl := tinysl.New()

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			_ = sl.Add(tinysl.Transient, nameServiceConstructor)

			wg.Done()
		}()

		wg.Add(1)
		go func() {
			_ = sl.Add(tinysl.Transient, nameServiceConstructor)

			wg.Done()
		}()

		wg.Wait()

		_, err := sl.ServiceLocator()
		Expect(err).Should(HaveOccurred())
	})

	It("should return error for circular dependencies", func() {
		_, err := tinysl.
			Add(tinysl.Transient, nameServiceConstructor).
			Add(tinysl.Transient, impostorConstructor).
			Add(tinysl.Transient, disguisedImpostorConstructor).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.ServiceBuilderError)))
		Expect(errors.Unwrap(err)).Should(BeAssignableToTypeOf(new(tinysl.CircularDependencyError)))
	})

	It("should return error for missing dependency", func() {
		_, err := tinysl.
			Add(tinysl.Transient, nameServiceConstructor).
			Add(tinysl.Transient, tableTimerConstructor).
			Add(tinysl.Transient, impostorConstructor).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.ServiceBuilderError)))
		Expect(errors.Unwrap(err)).Should(BeAssignableToTypeOf(new(tinysl.ConstructorNotFoundError)))
	})

	It("should return error for unsupported lifetime", func() {
		_, err := tinysl.
			Add("MyCustomLifetime", nameServiceConstructor).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(tinysl.LifetimeUnsupportedError("")))
	})

	It("should return error for wrong constructor type", func() {
		_, err := tinysl.
			Add(tinysl.Transient, "jsut random human made mistake").
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.BadConstructorError)))
		Expect(errors.Unwrap(err)).Should(BeAssignableToTypeOf(new(tinysl.ConstructorTemplateError)))
	})

	It("should return error for constructor returning wrong type", func() {
		badConstructor1 := func() error {
			return nil
		}

		badConstructor2 := func() (int, bool) {
			return 0, false
		}

		_, err := tinysl.
			Add(tinysl.Transient, badConstructor1).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.BadConstructorError)))
		Expect(errors.Unwrap(err)).Should(BeAssignableToTypeOf(new(tinysl.ConstructorTemplateError)))

		_, err = tinysl.
			Add(tinysl.Transient, badConstructor2).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.BadConstructorError)))
		Expect(errors.Unwrap(err)).Should(BeAssignableToTypeOf(new(tinysl.ConstructorTemplateError)))
	})

	It("should return error for constructor with context.Context not as a first argument", func() {
		badConstructor := func(nameService nameService, ctx context.Context) (*hero, error) {
			return &hero{name: nameService.Name()}, nil
		}

		_, err := tinysl.
			Add(tinysl.Transient, nameServiceConstructor).
			Add(tinysl.Transient, badConstructor).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.BadConstructorError)))
		Expect(errors.Unwrap(err)).Should(BeAssignableToTypeOf(new(tinysl.ConstructorTemplateError)))
	})

	It("should return first encountered error", func() {
		_, err := tinysl.
			Add(tinysl.Transient, "jsut random human made mistake").
			Add("MyCustomLifetime", nameServiceConstructor).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.BadConstructorError)))
		Expect(errors.Unwrap(err)).Should(BeAssignableToTypeOf(new(tinysl.ConstructorTemplateError)))
	})
})
