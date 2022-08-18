package tinysl_test

import (
	"reflect"

	"github.com/andriiyaremenko/tinysl"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("Constructor", func() {
	Context("T", func() {
		It("should return error if type argument is not a struct", func() {
			_, err := tinysl.T[int]()

			Expect(err).Should(HaveOccurred())
			Expect(err).Should(BeAssignableToTypeOf(new(tinysl.TError)))

			_, err = tinysl.T[string]()

			Expect(err).Should(HaveOccurred())
			Expect(err).Should(BeAssignableToTypeOf(new(tinysl.TError)))

			_, err = tinysl.T[*NameService]()

			Expect(err).Should(HaveOccurred())
			Expect(err).Should(BeAssignableToTypeOf(new(tinysl.TError)))

			_, err = tinysl.T[NameService]()

			Expect(err).Should(HaveOccurred())
			Expect(err).Should(BeAssignableToTypeOf(new(tinysl.TError)))
		})

		It("should work with a struct type argument", func() {
			constructor, err := tinysl.T[ServiceWithPublicFields]()

			Expect(err).ShouldNot(HaveOccurred())
			Expect(constructor.Type).
				To(Equal(reflect.TypeOf(ServiceWithPublicFields{})))
			Expect(constructor.Dependencies).
				To(Equal([]string{"tinysl_test.NameService"}))
		})
	})

	Context("P", func() {
		It("should return error if type argument is not a struct", func() {
			_, err := tinysl.P[int]()

			Expect(err).Should(HaveOccurred())
			Expect(err).Should(BeAssignableToTypeOf(new(tinysl.PError)))

			_, err = tinysl.P[string]()

			Expect(err).Should(HaveOccurred())
			Expect(err).Should(BeAssignableToTypeOf(new(tinysl.PError)))

			_, err = tinysl.P[*NameService]()

			Expect(err).Should(HaveOccurred())
			Expect(err).Should(BeAssignableToTypeOf(new(tinysl.PError)))

			_, err = tinysl.P[NameService]()

			Expect(err).Should(HaveOccurred())
			Expect(err).Should(BeAssignableToTypeOf(new(tinysl.PError)))
		})

		It("should work with a struct type argument", func() {
			constructor, err := tinysl.P[ServiceWithPublicFields]()

			Expect(err).ShouldNot(HaveOccurred())
			Expect(constructor.Type).
				To(Equal(reflect.TypeOf(&ServiceWithPublicFields{})))
			Expect(constructor.Dependencies).
				To(Equal([]string{"tinysl_test.NameService"}))
		})
	})

	Context("I", func() {
		It("should return error if T type argument is not a struct", func() {
			_, err := tinysl.I[int, int]()

			Expect(err).Should(HaveOccurred())
			Expect(err).Should(BeAssignableToTypeOf(new(tinysl.IError)))
			Expect(errors.Unwrap(err)).Should(MatchError(tinysl.ErrIWrongTType))

			_, err = tinysl.I[string, string]()

			Expect(err).Should(HaveOccurred())
			Expect(err).Should(BeAssignableToTypeOf(new(tinysl.IError)))
			Expect(errors.Unwrap(err)).Should(MatchError(tinysl.ErrIWrongTType))

			_, err = tinysl.I[NameService, *Hero]()

			Expect(err).Should(HaveOccurred())
			Expect(err).Should(BeAssignableToTypeOf(new(tinysl.IError)))
			Expect(errors.Unwrap(err)).Should(MatchError(tinysl.ErrIWrongTType))

			_, err = tinysl.I[NameService, NameService]()

			Expect(err).Should(HaveOccurred())
			Expect(err).Should(BeAssignableToTypeOf(new(tinysl.IError)))
			Expect(errors.Unwrap(err)).Should(MatchError(tinysl.ErrIWrongTType))
		})

		It("should return error if I type argument is not an interface", func() {
			_, err := tinysl.I[NameProvider, Impostor]()

			Expect(err).Should(HaveOccurred())
			Expect(err).Should(BeAssignableToTypeOf(new(tinysl.IError)))
			Expect(errors.Unwrap(err)).Should(MatchError(tinysl.ErrIWrongIType))
		})

		It("should return error if T type argument does not implement I type argument", func() {
			_, err := tinysl.I[NameService, Hero]()

			Expect(err).Should(HaveOccurred())
			Expect(err).Should(BeAssignableToTypeOf(new(tinysl.IError)))
			Expect(errors.Unwrap(err)).Should(MatchError(tinysl.ErrITDoesNotImplementI))
		})

		It("should work with when *T type argument implements I type argument", func() {
			constructor, err := tinysl.I[HelloService, ServiceWithPublicFields]()

			Expect(err).ShouldNot(HaveOccurred())
			Expect(constructor.Type).
				To(Equal(reflect.TypeOf(new(HelloService)).Elem()))
			Expect(constructor.Dependencies).
				To(Equal([]string{"tinysl_test.NameService"}))
		})
	})
})
