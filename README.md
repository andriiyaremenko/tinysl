# tinysl

[![GoDoc](https://img.shields.io/badge/pkg.go.dev-doc-blue)](http://pkg.go.dev/github.com/andriiyaremenko/tinysl)

This package provides simple abstraction to manage lifetime scope of services.
This package does not try to be another IOC container.
It was created because of need to share same instances of services among gorutines
within lifetime of a context.
PerContext lifetime scope was main reason to create this package,
other scopes were created for convenience.

To install tinysl:

```go
go get -u github.com/andriiyaremenko/tinysl
```

How to use:

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

sl := tinysl.New()

sl.Add(tinysl.PerContext, func(ctx context.Context) (MyService, error){
	// get your service instance
	return myservice("SomeService"), nil
})

sl.Add(tinysl.PerContext, func(ctx context.Context, myService MyService) (YourService, error){
	// get your service instance
	return yourservice(myService.Name()), nil
})

if err := sl.CanResolveDependencies(); err != nil {
	// handle missing dependency error
}

func MyRequestHandler(w http.ResponseWriter, req *http.Request) {
	var myService MyService

	service, err := sl.Get(req.Context(), &myService)
	if err != nil {
		// handle error
	}

	myService = service.(MyService)
	// use myService
}
```

Lifetime constants:

```go
tinysl.PerContext
tinysl.Singleton
tinysl.Transient
```

Constructor types that can be used:

```go
func(T1, T2, ...) (T, error)                // for PerContext, Transient and Singleton
func(context.Context, T1, T2, ...) (T, error) // for PerContext and Transient only
```

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
