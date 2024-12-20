# tinysl

[![GoDoc](https://img.shields.io/badge/pkg.go.dev-doc-blue)](http://pkg.go.dev/github.com/andriiyaremenko/tinysl)

This package provides a simple abstraction to manage lifetime scopes of services.
Its purpose is to help share instances of services among goroutines within context lifetime.

### To install tinysl:
`go get -u github.com/andriiyaremenko/tinysl`

### How to use:
```go
type MyService interface {
	SayHello() string
	Name() string
}

type YourService interface {
	ReplyHello() string
}

type myservice string
func (ms myservice) SayHello() string {
	return "Hello from " + string(myservice)
}

func (ms myservice) Name() string {
	return string(myservice)
}

type yourservice string
func (ms yourservice) ReplyHello() string {
	return "Hello to you too dear " + yourservice
}

sl, err := tinysl.
	Add(tinysl.PerContext, func(ctx context.Context) MyService {
		// get your service instance
		return myservice("SomeService")
	}).
	Add(tinysl.PerContext, func(ctx context.Context, myService MyService) (YourService, error){
		// get your service instance
		return yourservice(myService.Name()), nil
	}).
	ServiceLocator()
if err != nil {
	// handle error
}

func MyRequestHandler(w http.ResponseWriter, req *http.Request) {
	service, err := tinysl.Get[YourService](req.Context(), sl)
	if err != nil {
		// handle error
	}

	// use service
}

// or:
func GetMyRequestHandler(lazy tinysl.Lazy[YourService]) {
	func (w http.ResponseWriter, req *http.Request) {
		service := lazy(req.Context()

		// use service
	}
}

lazy := tinysl.Prepare[YourService](sl)
if err := sl.Err(); err != nil {
	// handle error
}
GetMyRequestHandler(lazy)
```
### Functions:
 * `tinysl.New`
 * `tinysl.Add`
 * `tinysl.Get`
 * `tinysl.MustGet`
 * `tinysl.Prepare`
 * `tinysl.DecorateHandler`
 * `tinysl.DecorateMiddleware`
 * `tinysl.SetLogger`

### Lifetime constants:
 * `tinysl.PerContext`
 * `tinysl.Singleton`
 * `tinysl.Transient`

### Constructor types that can be used:
 * `func(T1, T2, ...) [T|(T, error)]` - for Transient
 * `func(context.Context, T1, T2, ...) [T|(T, error)]` - for Transient
 * `func(T1, T2, ...) [T|(T, error)|(T, Cleanup, error)]` - for PerContext and Singleton
 * `func(context.Context, T1, T2, ...) [T|(T, error)|(T, Cleanup, error)]` - for PerContext only

### Public fields constructor
 * `tinysl.T[Type]` - would return `Type` instance with filled public fields using registered constructors.
 * `tinysl.P[Type]` - would return `*Type` instance with filled public fields using registered constructors.
 * `tinysl.I[Interface, Type]` - would return `Interface` implemented by `*Type` instance with filled public fields using registered constructors.
