package tinysl_test

import (
	"context"
	"sync"

	"errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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
		variadicConstructor := func(args ...any) (NameService, error) {
			return NameProvider("Bob"), nil
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
		badConstructor := func(nameService NameService, ctx context.Context) (*Hero, error) {
			return &Hero{name: nameService.Name()}, nil
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

	It("should return error if T is not called with a struct type argument", func() {
		_, err := tinysl.
			Add(tinysl.Transient, tinysl.T[int]).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.TError)))

		_, err = tinysl.
			Add(tinysl.Transient, tinysl.T[string]).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.TError)))

		_, err = tinysl.
			Add(tinysl.Transient, tinysl.T[*NameService]).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.TError)))

		_, err = tinysl.
			Add(tinysl.Transient, tinysl.T[NameService]).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.TError)))
	})

	It("should return error if T is called with type that was already added", func() {
		_, err := tinysl.
			Add(tinysl.Transient, func() (Hero, error) { return Hero{}, nil }).
			Add(tinysl.Transient, tinysl.T[Hero]).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.BadConstructorError)))
		Expect(errors.Unwrap(err)).Should(MatchError(tinysl.ErrDuplicateConstructor))
	})

	It("should work with T", func() {
		_, err := tinysl.
			Add(tinysl.Transient, nameServiceConstructor).
			Add(tinysl.Transient, tinysl.T[ServiceWithPublicFields]).
			ServiceLocator()

		Expect(err).ShouldNot(HaveOccurred())
	})

	It("should return error if P is not called with a struct type argument", func() {
		_, err := tinysl.
			Add(tinysl.Transient, tinysl.P[int]).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.PError)))

		_, err = tinysl.
			Add(tinysl.Transient, tinysl.P[string]).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.PError)))

		_, err = tinysl.
			Add(tinysl.Transient, tinysl.P[*NameService]).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.PError)))

		_, err = tinysl.
			Add(tinysl.Transient, tinysl.P[NameService]).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.PError)))
	})

	It("should return error if P is called with type that was already added", func() {
		_, err := tinysl.
			Add(tinysl.Transient, heroConstructor).
			Add(tinysl.Transient, tinysl.P[Hero]).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.BadConstructorError)))
		Expect(errors.Unwrap(err)).Should(MatchError(tinysl.ErrDuplicateConstructor))
	})

	It("should work with P", func() {
		_, err := tinysl.
			Add(tinysl.Transient, nameServiceConstructor).
			Add(tinysl.Transient, tinysl.P[ServiceWithPublicFields]).
			ServiceLocator()

		Expect(err).ShouldNot(HaveOccurred())
	})

	It("should return error if I is not called with T as a struct", func() {
		_, err := tinysl.
			Add(tinysl.Transient, tinysl.I[int, int]).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.IError)))
		Expect(errors.Unwrap(err)).Should(MatchError(tinysl.ErrIWrongTType))

		_, err = tinysl.
			Add(tinysl.Transient, tinysl.I[*int, int]).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.IError)))
		Expect(errors.Unwrap(err)).Should(MatchError(tinysl.ErrIWrongTType))

		_, err = tinysl.
			Add(tinysl.Transient, tinysl.I[string, string]).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.IError)))
		Expect(errors.Unwrap(err)).Should(MatchError(tinysl.ErrIWrongTType))

		_, err = tinysl.
			Add(tinysl.Transient, tinysl.I[*string, string]).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.IError)))
		Expect(errors.Unwrap(err)).Should(MatchError(tinysl.ErrIWrongTType))

		_, err = tinysl.
			Add(tinysl.Transient, tinysl.I[NameService, NameService]).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.IError)))
		Expect(errors.Unwrap(err)).Should(MatchError(tinysl.ErrIWrongTType))
	})

	It("should return error if I is called with I type argument that is not an interface", func() {
		_, err := tinysl.
			Add(tinysl.Transient, nameServiceConstructor).
			Add(tinysl.Transient, tinysl.I[NameProvider, Impostor]).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.IError)))
		Expect(errors.Unwrap(err)).Should(MatchError(tinysl.ErrIWrongIType))
	})

	It("should return error if I is called with T that does not implement I", func() {
		_, err := tinysl.
			Add(tinysl.Transient, nameServiceConstructor).
			Add(tinysl.Transient, tinysl.I[NameService, Hero]).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.IError)))
		Expect(errors.Unwrap(err)).Should(MatchError(tinysl.ErrITDoesNotImplementI))
	})

	It("should return error if I is called with I type that was already added", func() {
		_, err := tinysl.
			Add(tinysl.Transient, nameServiceConstructor).
			Add(tinysl.Transient, tinysl.I[NameService, Impostor]).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.BadConstructorError)))
		Expect(errors.Unwrap(err)).Should(MatchError(tinysl.ErrDuplicateConstructor))
	})

	It("should work with I", func() {
		_, err := tinysl.
			Add(tinysl.Transient, tinysl.I[NameService, Impostor]).
			ServiceLocator()

		Expect(err).ShouldNot(HaveOccurred())
	})
})
