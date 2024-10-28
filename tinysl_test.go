package tinysl_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/andriiyaremenko/tinysl"
)

var _ = Describe("Functions", func() {
	Context("MustGet", func() {
		It("should work", func() {
			sl, err := tinysl.
				Add(tinysl.PerContext, nameServiceConstructor).
				Add(tinysl.PerContext, tinysl.I[HelloService, ServiceWithPublicFields]).
				ServiceLocator()

			Expect(err).ShouldNot(HaveOccurred())

			ctx := context.TODO()
			ctx, cancel := context.WithCancel(ctx)

			defer cancel()

			service := tinysl.MustGet[HelloService](ctx, sl)
			Expect(service.Hello()).To(Equal("Hello Bob"))
		})

		It("should panic if constructor not found", func() {
			sl, err := tinysl.
				Add(tinysl.PerContext, nameServiceConstructor).
				Add(tinysl.PerContext, tinysl.I[HelloService, ServiceWithPublicFields]).
				ServiceLocator()

			Expect(err).ShouldNot(HaveOccurred())

			ctx := context.TODO()
			ctx, cancel := context.WithCancel(ctx)

			defer cancel()

			Expect(func() { tinysl.MustGet[*Hero](ctx, sl) }).To(Panic())
		})
	})

	Context("Prepare", func() {
		It("should work", func() {
			sl, err := tinysl.
				Add(tinysl.PerContext, nameServiceConstructor).
				Add(tinysl.PerContext, tinysl.I[HelloService, ServiceWithPublicFields]).
				ServiceLocator()

			Expect(err).ShouldNot(HaveOccurred())

			ctx := context.TODO()
			ctx, cancel := context.WithCancel(ctx)

			defer cancel()

			lazy := tinysl.Prepare[HelloService](sl)
			service := lazy(ctx)

			Expect(sl.Err()).ShouldNot(HaveOccurred())
			Expect(service.Hello()).To(Equal("Hello Bob"))
		})

		It("should report error", func() {
			sl, err := tinysl.
				Add(tinysl.PerContext, nameServiceConstructor).
				Add(tinysl.PerContext, tinysl.I[HelloService, ServiceWithPublicFields]).
				ServiceLocator()

			Expect(err).ShouldNot(HaveOccurred())

			ctx := context.TODO()
			ctx, cancel := context.WithCancel(ctx)

			defer cancel()

			lazy := tinysl.Prepare[*Hero](sl)

			Expect(sl.Err()).Should(HaveOccurred())
			Expect(sl.Err()).To(BeAssignableToTypeOf(&tinysl.ConstructorNotFoundError{}))
			Expect(func() { lazy(ctx) }).To(Panic())
		})
	})
	Context("Decorate", func() {
		It("should work", func() {
			sl, err := tinysl.
				Add(tinysl.PerContext, nameServiceConstructor).
				Add(tinysl.PerContext, tinysl.I[HelloService, ServiceWithPublicFields]).
				ServiceLocator()

			Expect(err).ShouldNot(HaveOccurred())

			ctx := context.TODO()
			_, cancel := context.WithCancel(ctx)

			defer cancel()

			type key string
			var nameKey key

			middleware := func(s NameService) func(http.Handler) http.Handler {
				return func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						ctx := context.WithValue(r.Context(), nameKey, s.Name())
						next.ServeHTTP(w, r.WithContext(ctx))
					})
				}
			}

			handler := tinysl.DecorateHandler(
				sl, func(h HelloService) http.HandlerFunc {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						Expect(r.Context().Value(nameKey)).NotTo(BeNil())
						Expect(r.Context().Value(nameKey)).To(Equal("Bob"))

						_, _ = w.Write([]byte(h.Hello()))
					})
				})

			Expect(sl.Err()).ShouldNot(HaveOccurred())

			server := httptest.NewServer(tinysl.DecorateMiddleware(sl, middleware)(handler))

			defer server.Close()

			resp, err := http.Get(server.URL)

			Expect(err).ShouldNot(HaveOccurred())

			defer resp.Body.Close()

			b, err := io.ReadAll(resp.Body)

			Expect(err).ShouldNot(HaveOccurred())

			hello := string(b)

			Expect(hello).To(Equal("Hello Bob"))

			server.Close()
		})

		It("should report error", func() {
			sl, err := tinysl.
				Add(tinysl.PerContext, nameServiceConstructor).
				Add(tinysl.PerContext, tinysl.I[HelloService, ServiceWithPublicFields]).
				ServiceLocator()

			Expect(err).ShouldNot(HaveOccurred())

			ctx := context.TODO()
			_, cancel := context.WithCancel(ctx)

			defer cancel()

			s := httptest.NewServer(tinysl.DecorateHandler(sl, func(h *Hero) http.Handler { return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}) }))

			defer s.Close()

			Expect(sl.Err()).Should(HaveOccurred())
			Expect(sl.Err()).To(BeAssignableToTypeOf(&tinysl.ConstructorNotFoundError{}))
		})
	})
})
