// This package provides simple abstraction to manage lifetime scope of services.
// This package does not try to be another IOC container.
// It was created because of need to share same instances of services among gorutines
// within lifetime of a context.
// PerContext lifetime scope was main reason to create this package,
// other scopes were created for convenience.
//
// To install tinysl:
// 	github.com/andriiyaremenko/tinysl
//
// How to use:
// type MyService interface {
// 	SayHello() string
// }
//
// type myservice string
// func (ms myservice) SayHello() string {
//	return "Hello from " + myservice
// }
//
// sl := tinysl.New()
//
// sl.Add(tinysl.PerContext, func(ctx context.Context) (MyService, error){
// 	// get your service instance
// 	return myservice{}, nil
// })
//
// func MyRequestHandler(w http.ResponseWriter, req *http.Request) {
// 	var myService MyService
//
// 	service, err := sl.Get(req.Context(), &myService)
// 	if err != nil {
// 		// process error
// 	}
//
// 	myService = service.(MyService)
// 	// use myService
//}
package tinysl