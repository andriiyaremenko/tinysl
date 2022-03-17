// This package provides simple abstraction to manage lifetime scope of services.
// This package does not try to be another IOC container.
// It was created because of need to share same instances of services among gorutines
// within lifetime of a context.
// PerContext lifetime scope was main reason to create this package,
// other scopes were created for convenience.
//
// To install tinysl:
// 	go get -u github.com/andriiyaremenko/tinysl
//
// How to use:
// 	type MyService interface {
// 		SayHello() string
// 		Name() string
// 	}
//
// 	type YourService interface {
// 		ReplyHello() string
// 	}
//
// 	type myservice string
// 	func (ms myservice) SayHello() string {
// 		return "Hello from " + string(myservice)
// 	}
//
// 	func (ms myservice) Name() string {
// 		return string(myservice)
// 	}
//
// 	type yourservice string
// 	func (ms yourservice) ReplyHello() string {
// 		return "Hello to you too dear " + yourservice
// 	}
//
// 	sl, err := tinysl.
// 		Add(tinysl.PerContext, func(ctx context.Context) (MyService, error){
// 			// get your service instance
// 			return myservice("SomeService"), nil
// 		}).
// 		Add(tinysl.PerContext, func(ctx context.Context, myService MyService) (YourService, error){
// 			// get your service instance
// 			return yourservice(myService.Name()), nil
// 		}).
//		ServiceLocator()
//	if err != nil {
//		// handle error
//	}
//
// 	func MyRequestHandler(w http.ResponseWriter, req *http.Request) {
// 		service, err := tinysl.Get[YourService](req.Context(), sl)
// 		if err != nil {
// 			// handle error
// 		}
//
// 		// use service
// 	}
//
// Lifetime constants:
// 	tinysl.PerContext
// 	tinysl.Singleton
// 	tinysl.Transient
//
// Constructor types that can be used:
// 	func(T1, T2, ...) (T, error)                // for PerContext, Transient and Singleton
// 	func(context.Context, T1, T2, ...) (T, error) // for PerContext and Transient only
package tinysl
