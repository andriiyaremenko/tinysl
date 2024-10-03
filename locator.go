// NOTE: building dependency graph for PerContext for every service
// and assembling them within single call to (*locator).getSingleton
// proved to be ineffective compared to existing implementation (memory and speed wise).
// Turns out calling sync.Map for every context with additional data structures that comes with dependency graph
// is more expensive compared to calling sync.Map for every PerContext service without additional data structures.
package tinysl

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"
	"time"
)

var _ ServiceLocator = new(locator)

type cleanupRecord struct {
	ctx context.Context
	fn  Cleanup
}

// worker to handle singletons cleanup before application exit
func singletonCleanupWorker(
	ctx context.Context, cancel context.CancelFunc,
	singletonsCleanupCh <-chan Cleanup, wg *sync.WaitGroup,
) {
	var cleanup Cleanup = func() {}

loop:
	for {
		select {
		case fn := <-singletonsCleanupCh:
			oldFn := cleanup
			cleanup = func() {
				fn()
				oldFn()
			}
		case <-ctx.Done():
			cleanup.CallWithRecovery(Singleton)
			break loop
		}
	}

	wg.Wait()
	cancel()
}

// worker to handle per-context cleanups
func perContextCleanupWorker(ctx context.Context, perContextCleanupCh <-chan cleanupRecord, wg *sync.WaitGroup) {
	cleanups := make(map[uintptr]Cleanup)
	ctxList := []context.Context{}
	nextCtx := context.Background()
	replaceNextContext := true
	ticker := time.NewTicker(time.Second)

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		default:
		}

		select {
		case rec := <-perContextCleanupCh:
			pt := reflect.ValueOf(rec.ctx).Pointer()
			fn, ok := cleanups[pt]

			if ok {
				oldFn := fn
				fn = func() {
					rec.fn()
					oldFn()
				}
			} else {
				fn = rec.fn
			}

			cleanups[pt] = fn

			if replaceNextContext {
				nextCtx = rec.ctx
				replaceNextContext = false
			}

			ctxList = append(ctxList, rec.ctx)
		case <-nextCtx.Done():
			pt := reflect.ValueOf(nextCtx).Pointer()
			if fn, ok := cleanups[pt]; ok {
				fn.CallWithRecovery(PerContext)
			}

			delete(cleanups, pt)

			if len(ctxList) == 0 {
				nextCtx = context.Background()
				replaceNextContext = true
			} else {
				nextCtx = ctxList[0]
				ctxList = ctxList[1:]
			}
		case <-ticker.C:
			if len(ctxList) > 1 {
				select {
				case <-nextCtx.Done():
					continue loop
				default:
				}

				for i := range ctxList {
					j := rand.IntN(i + 1)
					ctxList[i], ctxList[j] = ctxList[j], ctxList[i]
				}

				nextCtx = ctxList[0]
			}
		case <-ctx.Done():
			break loop
		}
	}

	ticker.Stop()

	for _, fn := range cleanups {
		fn.CallWithRecovery(PerContext)
	}

	wg.Done()
}

func newLocator(ctx context.Context, constructors map[string]*record, size uint) ServiceLocator {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	singletonsCleanupCh := make(chan Cleanup)
	perContextCleanupCh := make(chan cleanupRecord)
	var wg sync.WaitGroup

	go singletonCleanupWorker(ctx, cancel, singletonsCleanupCh, &wg)

	for i := uint(0); i < size; i++ {
		wg.Add(1)
		go perContextCleanupWorker(ctx, perContextCleanupCh, &wg)
	}

	perCtxIDs := make([]uintptr, 0)
	for _, rec := range constructors {
		if rec.lifetime == PerContext {
			perCtxIDs = append(perCtxIDs, rec.id)
		}
	}

	return &locator{
		constructors:        constructors,
		perContext:          newContextInstances(perCtxIDs),
		singletonsCleanupCh: singletonsCleanupCh,
		perContextCleanUpCh: perContextCleanupCh,
	}
}

type locator struct {
	err                 error
	perContext          *contextInstances
	constructors        map[string]*record
	singletonsCleanupCh chan<- Cleanup
	perContextCleanUpCh chan<- cleanupRecord
	sMu                 sync.Map
	pcMu                sync.Map
	singletons          sync.Map
	errRMu              sync.RWMutex
}

func (l *locator) EnsureAvailable(serviceName string) {
	for key := range l.constructors {
		if key == serviceName {
			return
		}
	}

	l.errRMu.Lock()
	l.err = newConstructorNotFoundError(serviceName)
	l.errRMu.Unlock()
}

func (l *locator) Err() error {
	l.errRMu.RLock()
	defer l.errRMu.RUnlock()

	return l.err
}

func (l *locator) Get(ctx context.Context, serviceName string) (any, error) {
	record, ok := l.constructors[serviceName]

	if !ok {
		return nil, newConstructorNotFoundError(serviceName)
	}

	switch record.lifetime {
	case Singleton:
		return l.getSingleton(ctx, record)
	case PerContext:
		return l.getPerContext(ctx, record, serviceName)
	case Transient:
		s, _, err := l.get(ctx, record)
		return s, err
	default:
		panic(fmt.Errorf(
			"broken record %s: %w",
			record.typeName,
			LifetimeUnsupportedError(record.lifetime.String())),
		)
	}
}

func (l *locator) get(ctx context.Context, record *record) (service any, cleanups Cleanup, err error) {
	defer func() {
		if rp := recover(); rp != nil {
			err = newServiceBuilderError(
				newConstructorError(fmt.Errorf("recovered from panic: %v", rp)),
				record.lifetime,
				record.typeName,
			)
		}
	}()

	constructor := record.constructor
	fn := reflect.ValueOf(constructor)
	args := make([]reflect.Value, 0, 1)

	for i, dep := range record.dependencies {
		if i == 0 && dep == contextDepName {
			args = append(args, reflect.ValueOf(ctx))
			continue
		}

		service, err := l.Get(ctx, dep)
		if err != nil {
			return nil, nil, err
		}

		args = append(args, reflect.ValueOf(service))
	}

	values := fn.Call(args)

	if record.constructorType == onlyService && len(values) != 1 ||
		record.constructorType == withError && len(values) != 2 ||
		record.constructorType == withErrorAndCleanUp && len(values) != 3 {
		return nil, nil, newServiceBuilderError(
			newConstructorError(newUnexpectedResultError(values)),
			record.lifetime,
			record.typeName,
		)
	}

	switch record.constructorType {
	case onlyService:
		service := values[0].Interface()
		return service, func() {}, nil
	case withError:
		serviceV, errV := values[0], values[1]
		if err, ok := (errV.Interface()).(error); ok && err != nil {
			return nil, nil, newServiceBuilderError(
				newConstructorError(err),
				record.lifetime,
				record.typeName,
			)
		}

		service := serviceV.Interface()

		return service, func() {}, nil
	case withErrorAndCleanUp:
		serviceV, cleanUpV, errV := values[0], values[1], values[2]
		if err, ok := (errV.Interface()).(error); ok && err != nil {
			return nil, nil, newServiceBuilderError(
				newConstructorError(err),
				record.lifetime,
				record.typeName,
			)
		}

		service := serviceV.Interface()
		cleanUp := cleanUpV.Interface()

		return service, cleanUp.(func()), nil
	default:
		return nil, nil, newServiceBuilderError(
			newConstructorUnsupportedError(
				fn.Type(),
				record.lifetime,
			),
			record.lifetime,
			record.typeName,
		)
	}
}

func (l *locator) getSingleton(ctx context.Context, record *record) (any, error) {
	mu, ok := l.sMu.LoadOrStore(record.id, new(sync.Mutex))

	mu.(*sync.Mutex).Lock()
	defer mu.(*sync.Mutex).Unlock()

	servicePtr, ok := l.singletons.Load(record.id)
	if ok {
		return *servicePtr.(*any), nil
	}

	service, cleanUp, err := l.get(ctx, record)
	if err != nil {
		return nil, err
	}

	go func() { l.singletonsCleanupCh <- cleanUp }()

	l.singletons.Store(record.id, &service)

	return service, nil
}

func (l *locator) getPerContext(ctx context.Context, record *record, serviceName string) (any, error) {
	if ctx == nil {
		return nil, newServiceBuilderError(ErrNilContext, record.lifetime, serviceName)
	}

	if err := ctx.Err(); err != nil {
		return nil, newServiceBuilderError(err, record.lifetime, serviceName)
	}

	ctxKey := reflect.ValueOf(ctx).Pointer()
	scope, ok := l.perContext.get(ctxKey, record.id)

	scope.lock()
	defer scope.unlock()

	if !scope.empty() {
		return *scope.value, nil
	}

	service, cleanUp, err := l.get(ctx, record)
	if err != nil {
		return nil, err
	}

	if !ok {
		go func() {
			l.perContextCleanUpCh <- cleanupRecord{
				ctx: ctx,
				fn: func() {
					cleanUp()
					l.perContext.delete(ctxKey)
				},
			}
		}()
	} else {
		go func() { l.perContextCleanUpCh <- cleanupRecord{ctx: ctx, fn: cleanUp} }()
	}

	scope.value = &service

	return service, nil
}
