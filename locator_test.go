package tinysl_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/goleak"

	"github.com/andriiyaremenko/tinysl"
)

var _ = Describe("ServiceLocator", func() {
	var ctx context.Context

	BeforeEach(func() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(context.Background())

		DeferCleanup(func() {
			cancel()
		})
	})

	It("should return new instance every time for Transient", func() {
		sl, err := tinysl.
			Add(tinysl.Transient, nameServiceConstructor).
			Add(tinysl.Transient, heroConstructor).
			ServiceLocator()

		Expect(err).ShouldNot(HaveOccurred())

		hero1, err := tinysl.Get[*Hero](ctx, sl)

		Expect(err).ShouldNot(HaveOccurred())

		hero2, err := tinysl.Get[*Hero](ctx, sl)

		Expect(err).ShouldNot(HaveOccurred())
		Expect(hero1).NotTo(BeIdenticalTo(hero2))
	})

	It("should return same instance for same context for PerContext", func() {
		sl, err := tinysl.
			Add(tinysl.PerContext, nameServiceConstructor).
			Add(tinysl.PerContext, heroConstructor).
			ServiceLocator()

		Expect(err).ShouldNot(HaveOccurred())

		hero1, err := tinysl.Get[*Hero](ctx, sl)

		Expect(err).ShouldNot(HaveOccurred())

		hero2, err := tinysl.Get[*Hero](ctx, sl)

		Expect(err).ShouldNot(HaveOccurred())
		Expect(hero1).To(BeIdenticalTo(hero2))
	})

	It("should return new instance for different context for PerContext", func() {
		sl, err := tinysl.
			Add(tinysl.PerContext, nameServiceConstructor).
			Add(tinysl.PerContext, heroConstructor).
			ServiceLocator()

		Expect(err).ShouldNot(HaveOccurred())

		ctx1 := context.TODO()
		ctx1, cancel1 := context.WithCancel(ctx1)
		ctx2, cancel2 := context.WithCancel(ctx1)

		defer cancel1()
		defer cancel2()

		hero1, err := tinysl.Get[*Hero](ctx1, sl)

		Expect(err).ShouldNot(HaveOccurred())

		hero2, err := tinysl.Get[*Hero](ctx2, sl)

		Expect(err).ShouldNot(HaveOccurred())
		Expect(hero1).NotTo(BeIdenticalTo(hero2))
	})

	It("should return error for cancelled context for PerContext", func() {
		sl, err := tinysl.
			Add(tinysl.PerContext, nameServiceConstructor).
			Add(tinysl.PerContext, heroConstructor).
			ServiceLocator()

		Expect(err).ShouldNot(HaveOccurred())

		ctx := context.TODO()
		ctx, cancel := context.WithCancel(ctx)
		cancel()

		_, err = tinysl.Get[*Hero](ctx, sl)

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.ServiceBuilderError)))
		Expect(errors.Unwrap(err)).Should(MatchError(context.Canceled))
	})

	It("should return error for nil context for PerContext", func() {
		sl, err := tinysl.
			Add(tinysl.PerContext, nameServiceConstructor).
			Add(tinysl.PerContext, heroConstructor).
			ServiceLocator()

		Expect(err).ShouldNot(HaveOccurred())

		_, err = tinysl.Get[*Hero](nil, sl)

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.ServiceBuilderError)))
		Expect(errors.Unwrap(err)).Should(MatchError(tinysl.ErrNilContext))
	})

	It("should always return the same instance for Singleton", func() {
		sl, err := tinysl.
			Add(tinysl.Singleton, nameServiceConstructor).
			Add(tinysl.Singleton, heroConstructor).
			ServiceLocator()
		Expect(err).ShouldNot(HaveOccurred())

		ctx1, cancel := context.WithCancel(ctx)

		defer cancel()

		hero1, err := tinysl.Get[*Hero](ctx, sl)

		Expect(err).ShouldNot(HaveOccurred())

		hero2, err := tinysl.Get[*Hero](ctx1, sl)

		Expect(err).ShouldNot(HaveOccurred())
		Expect(hero1).To(BeIdenticalTo(hero2))
	})

	It("should manage constructor dependencies", func() {
		sl, err := tinysl.
			Add(tinysl.Transient, nameServiceConstructor).
			Add(tinysl.Transient, tableTimerConstructor).
			Add(tinysl.Transient, heroConstructor).
			Add(tinysl.Transient, impostorConstructor).
			ServiceLocator()

		Expect(err).ShouldNot(HaveOccurred())

		impostor, err := tinysl.Get[*Impostor](ctx, sl)

		Expect(err).ShouldNot(HaveOccurred())
		Expect(impostor.Announce()).To(Equal("Bob is our hero!"))

		hero, err := tinysl.Get[*Hero](ctx, sl)

		Expect(err).ShouldNot(HaveOccurred())
		Expect(hero.Announce()).To(Equal("Bob is our hero!"))

		nameService, err := tinysl.Get[NameService](ctx, sl)

		Expect(err).ShouldNot(HaveOccurred())
		Expect(nameService.Name()).To(Equal("Bob"))
	})

	It("should return error on missing constructor", func() {
		sl, err := tinysl.
			Add(tinysl.Transient, nameServiceConstructor).
			Add(tinysl.Transient, tableTimerConstructor).
			Add(tinysl.Transient, heroConstructor).
			ServiceLocator()

		Expect(err).ShouldNot(HaveOccurred())

		_, err = tinysl.Get[*Impostor](ctx, sl)

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.ConstructorNotFoundError)))
	})

	It("should be tread-safe for Singleton", func() {
		for i := 5_000; i > 0; i-- {
			sl, err := tinysl.
				Add(tinysl.Singleton, nameServiceConstructor).
				Add(tinysl.Singleton, heroConstructor).
				ServiceLocator()

			Expect(err).ShouldNot(HaveOccurred())

			ctx1, cancel1 := context.WithCancel(ctx)
			ctx2, cancel2 := context.WithCancel(ctx)

			var hero1, hero2, hero3 *Hero
			var err1, err2, err3 error
			var wg sync.WaitGroup

			wg.Add(1)
			go func() {
				defer GinkgoRecover()

				hero1, err1 = tinysl.Get[*Hero](ctx1, sl)

				Expect(err1).ShouldNot(HaveOccurred())
				wg.Done()
			}()

			wg.Add(1)
			go func() {
				defer GinkgoRecover()

				hero2, err2 = tinysl.Get[*Hero](ctx2, sl)

				Expect(err2).ShouldNot(HaveOccurred())
				wg.Done()
			}()

			wg.Add(1)
			go func() {
				defer GinkgoRecover()

				hero3, err3 = tinysl.Get[*Hero](ctx1, sl)

				Expect(err3).ShouldNot(HaveOccurred())
				wg.Done()
			}()

			wg.Add(1)
			go func() {
				defer GinkgoRecover()

				_, err = tinysl.Get[NameService](ctx1, sl)

				Expect(err).ShouldNot(HaveOccurred())
				wg.Done()
			}()

			wg.Wait()
			cancel1()
			cancel2()
			Expect(hero1).NotTo(BeNil())
			Expect(hero2).NotTo(BeNil())
			Expect(hero3).NotTo(BeNil())
			Expect(hero1).To(BeIdenticalTo(hero2))
			Expect(hero3).To(BeIdenticalTo(hero2))
			Expect(hero1).To(BeIdenticalTo(hero3))
		}
	})

	It("should be tread-safe for PerContext", func() {
		for i := 5_000; i > 0; i-- {
			sl, err := tinysl.
				Add(tinysl.PerContext, nameServiceConstructor).
				Add(tinysl.PerContext, heroConstructor).
				ServiceLocator()

			Expect(err).ShouldNot(HaveOccurred())

			ctx1, cancel1 := context.WithCancel(ctx)
			ctx2, cancel2 := context.WithCancel(ctx)

			var hero1, hero2, hero3 *Hero
			var err1, err2, err3 error
			var wg sync.WaitGroup

			wg.Add(1)
			go func() {
				defer GinkgoRecover()

				hero1, err1 = tinysl.Get[*Hero](ctx1, sl)

				Expect(err1).ShouldNot(HaveOccurred())
				wg.Done()
			}()

			wg.Add(1)
			go func() {
				defer GinkgoRecover()

				hero2, err2 = tinysl.Get[*Hero](ctx2, sl)

				Expect(err2).ShouldNot(HaveOccurred())
				wg.Done()
			}()

			wg.Add(1)
			go func() {
				defer GinkgoRecover()

				hero3, err3 = tinysl.Get[*Hero](ctx1, sl)

				Expect(err3).ShouldNot(HaveOccurred())
				wg.Done()
			}()

			wg.Add(1)
			go func() {
				defer GinkgoRecover()

				_, err = tinysl.Get[NameService](ctx1, sl)

				Expect(err).ShouldNot(HaveOccurred())
				wg.Done()
			}()

			wg.Wait()
			cancel1()
			cancel2()
			Expect(hero1).NotTo(BeNil())
			Expect(hero2).NotTo(BeNil())
			Expect(hero3).NotTo(BeNil())
			Expect(hero1).NotTo(BeIdenticalTo(hero2))
			Expect(hero3).NotTo(BeIdenticalTo(hero2))
			Expect(hero1).To(BeIdenticalTo(hero3))
		}
	})

	It("should not leak goroutines", func() {
		for i := 10; i > 0; i-- {
			sl, err := tinysl.
				Add(tinysl.PerContext, nameServiceConstructor).
				Add(tinysl.PerContext, heroConstructor).
				ServiceLocator()

			Expect(err).ShouldNot(HaveOccurred())

			ctx1, cancel1 := context.WithCancel(ctx)
			ctx2, cancel2 := context.WithCancel(ctx)

			var hero1, hero2, hero3 *Hero

			hero1, err = tinysl.Get[*Hero](ctx1, sl)

			Expect(err).ShouldNot(HaveOccurred())

			hero2, err = tinysl.Get[*Hero](ctx2, sl)

			Expect(err).ShouldNot(HaveOccurred())

			hero3, err = tinysl.Get[*Hero](ctx1, sl)

			Expect(err).ShouldNot(HaveOccurred())

			_, err = tinysl.Get[NameService](ctx1, sl)

			Expect(err).ShouldNot(HaveOccurred())

			cancel1()
			cancel2()

			Expect(hero1).NotTo(BeNil())
			Expect(hero2).NotTo(BeNil())
			Expect(hero3).NotTo(BeNil())
			Expect(hero1).NotTo(BeIdenticalTo(hero2))
			Expect(hero3).NotTo(BeIdenticalTo(hero2))
			Expect(hero1).To(BeIdenticalTo(hero3))
		}

		time.Sleep(time.Millisecond)
		err := goleak.Find(
			goleak.
				IgnoreTopFunction(
					"github.com/onsi/ginkgo/v2/internal.(*Suite).runNode",
				),
			goleak.
				IgnoreTopFunction(
					"github.com/onsi/ginkgo/v2/internal/interrupt_handler.(*InterruptHandler).registerForInterrupts.func2",
				),
			goleak.
				IgnoreAnyFunction(
					"github.com/onsi/ginkgo/v2/internal.RegisterForProgressSignal.func1",
				),
			goleak.
				IgnoreAnyFunction(
					"github.com/andriiyaremenko/tinysl.perContextCleanupWorker",
				),
			goleak.
				IgnoreAnyFunction(
					"github.com/andriiyaremenko/tinysl.singletonCleanupWorker",
				),
			goleak.
				IgnoreAnyFunction(
					"os/signal.NotifyContext.func1",
				),
		)

		Expect(err).ShouldNot(HaveOccurred())
	})

	It("should return error if constructor returned error", func() {
		errConstructor := func() (NameService, error) {
			return nil, errors.New("some unfortunate error")
		}
		sl, err := tinysl.
			Add(tinysl.PerContext, errConstructor).
			Add(tinysl.PerContext, heroConstructor).
			ServiceLocator()

		Expect(err).ShouldNot(HaveOccurred())

		_, err = tinysl.Get[*Hero](ctx, sl)

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.ServiceBuilderError)))

		err = errors.Unwrap(err)

		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.ConstructorError)))
		Expect(errors.Unwrap(err)).Should(MatchError("some unfortunate error"))
	})

	It("should work with T", func() {
		sl, err := tinysl.
			Add(tinysl.PerContext, nameServiceConstructor).
			Add(tinysl.PerContext, tinysl.T[ServiceWithPublicFields]).
			ServiceLocator()

		Expect(err).ShouldNot(HaveOccurred())

		service, err := tinysl.Get[ServiceWithPublicFields](ctx, sl)

		Expect(err).ShouldNot(HaveOccurred())

		Expect(service.SomeProperty()).To(BeEmpty())
		Expect(service.Name()).To(Equal("Bob"))
	})

	It("should work with P", func() {
		sl, err := tinysl.
			Add(tinysl.PerContext, nameServiceConstructor).
			Add(tinysl.PerContext, tinysl.P[ServiceWithPublicFields]).
			ServiceLocator()

		Expect(err).ShouldNot(HaveOccurred())

		service, err := tinysl.Get[*ServiceWithPublicFields](ctx, sl)

		Expect(err).ShouldNot(HaveOccurred())

		Expect(service.SomeProperty()).To(BeEmpty())
		Expect(service.Name()).To(Equal("Bob"))
	})

	It("should work with I", func() {
		sl, err := tinysl.
			Add(tinysl.PerContext, nameServiceConstructor).
			Add(tinysl.PerContext, tinysl.I[HelloService, ServiceWithPublicFields]).
			ServiceLocator()

		Expect(err).ShouldNot(HaveOccurred())

		service, err := tinysl.Get[HelloService](ctx, sl)

		Expect(err).ShouldNot(HaveOccurred())

		Expect(service.Hello()).To(Equal("Hello Bob"))
		Expect(service.(*ServiceWithPublicFields).SomeProperty()).To(BeEmpty())
	})

	It("should use cleanup function for PerContext", func() {
		cleaned := make(chan struct{})
		sl, err := tinysl.
			Add(tinysl.PerContext, nameServiceConstructor).
			Add(tinysl.PerContext, heroConstructorWithCleanup(func() { close(cleaned) })).
			ServiceLocator()
		Expect(err).ShouldNot(HaveOccurred())

		ctx, cancel := context.WithCancel(ctx)
		_, err = tinysl.Get[*Hero](ctx, sl)

		Expect(err).ShouldNot(HaveOccurred())

		time.Sleep(time.Millisecond)
		cancel()
		Eventually(cleaned).Should(BeClosed())
	})

	It("should use cleanup function for Singleton", func() {
		appCtx := context.Background()
		appCtx, cancel := context.WithCancel(appCtx)
		cleaned := make(chan struct{})
		sl, err := tinysl.
			New(tinysl.WithSingletonCleanupContext(appCtx)).
			Add(tinysl.Singleton, nameServiceConstructor).
			Add(tinysl.Singleton, heroConstructorWithCleanup(func() { close(cleaned) })).
			ServiceLocator()

		Expect(err).ShouldNot(HaveOccurred())

		_, err = tinysl.Get[*Hero](ctx, sl)

		Expect(err).ShouldNot(HaveOccurred())

		time.Sleep(time.Millisecond)
		cancel()
		Eventually(cleaned).Should(BeClosed())
	})

	It("should handle panic", func() {
		sl, err := tinysl.
			Add(tinysl.PerContext, nameServiceConstructor).
			Add(tinysl.PerContext, scaredHeroConstructorWithCleanup).
			ServiceLocator()
		Expect(err).ShouldNot(HaveOccurred())

		_, err = tinysl.Get[*Hero](ctx, sl)

		Expect(err).Should(HaveOccurred())
		Expect(err).Should(BeAssignableToTypeOf(new(tinysl.ServiceBuilderError)))
		Expect(errors.Unwrap(err)).Should(MatchError(fmt.Errorf("recovered from panic: scared")))
	})

	It("should handle panic during cleanup function for PerContext", func() {
		cleaned := make(chan struct{})
		sl, err := tinysl.
			Add(tinysl.PerContext, nameServiceConstructor).
			Add(tinysl.PerContext, heroConstructorWithCleanup(func() { close(cleaned); panic("oops") })).
			ServiceLocator()
		Expect(err).ShouldNot(HaveOccurred())

		ctx, cancel := context.WithCancel(ctx)
		_, err = tinysl.Get[*Hero](ctx, sl)

		Expect(err).ShouldNot(HaveOccurred())

		time.Sleep(time.Millisecond)
		cancel()
		Eventually(cleaned).Should(BeClosed())
	})

	It("should handle panic during cleanup function for Singleton", func() {
		appCtx := context.Background()
		appCtx, cancel := context.WithCancel(appCtx)

		cleaned := make(chan struct{})
		sl, err := tinysl.
			New(tinysl.WithSingletonCleanupContext(appCtx)).
			Add(tinysl.Singleton, nameServiceConstructor).
			Add(tinysl.Singleton, heroConstructorWithCleanup(func() { close(cleaned); panic("oops") })).
			ServiceLocator()

		Expect(err).ShouldNot(HaveOccurred())

		_, err = tinysl.Get[*Hero](ctx, sl)

		Expect(err).ShouldNot(HaveOccurred())

		time.Sleep(time.Millisecond)
		cancel()
		Eventually(cleaned).Should(BeClosed())
	})

	It("should work with constructor without error", func() {
		sl, err := tinysl.
			Add(tinysl.Singleton, func() NameProvider { return NameProvider("Bob") }).
			ServiceLocator()

		Expect(err).ShouldNot(HaveOccurred())

		s, err := tinysl.Get[NameProvider](ctx, sl)

		Expect(err).ShouldNot(HaveOccurred())
		Expect(s).ShouldNot(BeNil())
	})
})
