package tinysl_test

import (
	"context"
	"sync"

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
			Add(tinysl.Singleton, tableTimerConstructor).
			ServiceLocator()
		Expect(err).Should(HaveOccurred())
	})

	It("should register PerContext", func() {
		_, err := tinysl.Add(tinysl.PerContext, nameProviderConstructor).ServiceLocator()
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("should register PerContext constructor dependant on context.Context", func() {
		_, err := tinysl.
			Add(tinysl.PerContext, tableTimerConstructor).
			ServiceLocator()
		Expect(err).Should(HaveOccurred())
	})

	It("should register Transient", func() {
		_, err := tinysl.Add(tinysl.Transient, nameProviderConstructor).ServiceLocator()
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("should register Transient constructor dependant on context.Context", func() {
		_, err := tinysl.
			Add(tinysl.Transient, tableTimerConstructor).
			ServiceLocator()
		Expect(err).Should(HaveOccurred())
	})

	It("should not allow add duplicate services for same lifetime", func() {
		_, err := tinysl.
			Add(tinysl.Transient, nameProviderConstructor).
			Add(tinysl.Transient, nameProviderConstructor).
			ServiceLocator()
		Expect(err).Should(HaveOccurred())
		Expect(err).
			Should(
				MatchError(
					"ServiceLocator has already registered constructor for tinysl_test.nameProvider - func() (tinysl_test.nameProvider, error)",
				),
			)
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
		Expect(err).
			Should(
				MatchError(
					"variadic function as constructor is not supported, got func(...interface {}) (tinysl_test.nameService, error)",
				),
			)
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
		Expect(err).
			Should(
				MatchError(
					"circular dependency in func(*tinysl_test.impostor) (*tinysl_test.hero, error): *tinysl_test.hero depends on *tinysl_test.impostor",
				),
			)
	})

	It("should return error for missing dependency", func() {
		_, err := tinysl.
			Add(tinysl.Transient, nameServiceConstructor).
			Add(tinysl.Transient, tableTimerConstructor).
			Add(tinysl.Transient, impostorConstructor).
			ServiceLocator()
		Expect(err).Should(HaveOccurred())
		Expect(err).
			Should(
				MatchError(
					"*tinysl_test.impostor has unregistered dependency: constructor for *tinysl_test.hero not found",
				),
			)
	})

	It("should return error for unsupported lifetime", func() {
		_, err := tinysl.
			Add("MyCustomLifetime", nameServiceConstructor).
			ServiceLocator()
		Expect(err).Should(HaveOccurred())
		Expect(err).Should(MatchError("MyCustomLifetime Lifetime is unsupported"))
	})

	It("should return error for wrong constructor type", func() {
		_, err := tinysl.
			Add(tinysl.Transient, "jsut random human made mistake").
			ServiceLocator()
		Expect(err).Should(HaveOccurred())
		Expect(err).
			Should(
				MatchError(
					"constructor should be of type func(T1, T2, ...) (T, error) | func(context.Context, T1, T2, ...) (T, error) for Transient, got string",
				),
			)
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
		Expect(err).
			Should(
				MatchError(
					"constructor should be of type func(T1, T2, ...) (T, error) | func(context.Context, T1, T2, ...) (T, error) for Transient, got func() error",
				),
			)

		_, err = tinysl.
			Add(tinysl.Transient, badConstructor2).
			ServiceLocator()

		Expect(err).Should(HaveOccurred())
		Expect(err).
			Should(
				MatchError(
					"constructor should be of type func(T1, T2, ...) (T, error) | func(context.Context, T1, T2, ...) (T, error) for Transient, got func() (int, bool)",
				),
			)
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
		Expect(err).
			Should(
				MatchError(
					"constructor should be of type func(T1, T2, ...) (T, error) | func(context.Context, T1, T2, ...) (T, error) for Transient, got func(tinysl_test.nameService, context.Context) (*tinysl_test.hero, error)",
				),
			)
	})

	It("should return first encountered error", func() {
		_, err := tinysl.
			Add(tinysl.Transient, "jsut random human made mistake").
			Add("MyCustomLifetime", nameServiceConstructor).
			ServiceLocator()
		Expect(err).Should(HaveOccurred())
		Expect(err).
			Should(
				MatchError(
					"constructor should be of type func(T1, T2, ...) (T, error) | func(context.Context, T1, T2, ...) (T, error) for Transient, got string",
				),
			)
	})

})
