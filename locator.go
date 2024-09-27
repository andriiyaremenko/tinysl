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
				oldFn()
				fn()
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
	cleanups := make(map[string]Cleanup)
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
			key := getPerContextKey(rec.ctx, "")
			fn, ok := cleanups[key]

			if ok {
				oldFn := fn
				fn = func() {
					oldFn()
					rec.fn()
				}
			} else {
				fn = rec.fn
			}

			cleanups[key] = fn

			if replaceNextContext {
				nextCtx = rec.ctx
				replaceNextContext = false
			}

			ctxList = append(ctxList, rec.ctx)
		case <-nextCtx.Done():
			key := getPerContextKey(nextCtx, "")
			if fn, ok := cleanups[key]; ok {
				fn.CallWithRecovery(PerContext)
			}

			delete(cleanups, key)

			if len(ctxList) == 0 {
				nextCtx = context.Background()
				replaceNextContext = true
			} else {
				nextCtx = ctxList[0]
				ctxList = ctxList[1:]
			}
		case <-ticker.C:
			if len(ctxList) > 1 {
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

func newLocator(ctx context.Context, constructors map[string]record, size uint) ServiceLocator {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	singletonsCleanupCh := make(chan Cleanup)
	perContextCleanupCh := make(chan cleanupRecord)
	var wg sync.WaitGroup

	go singletonCleanupWorker(ctx, cancel, singletonsCleanupCh, &wg)

	for i := uint(0); i < size; i++ {
		wg.Add(1)
		go perContextCleanupWorker(ctx, perContextCleanupCh, &wg)
	}

	return &locator{
		constructors:        constructors,
		singletons:          newInstances(),
		perContext:          newContextInstances(),
		sMu:                 make(map[string]*sync.Mutex),
		pcMu:                make(map[string]*sync.Mutex),
		singletonsCleanupCh: singletonsCleanupCh,
		perContextCleanUpCh: perContextCleanupCh,
	}
}

type locator struct {
	err                 error
	sMu                 map[string]*sync.Mutex
	pcMu                map[string]*sync.Mutex
	singletons          *instances
	perContext          *contextInstances
	constructors        map[string]record
	singletonsCleanupCh chan<- Cleanup
	perContextCleanUpCh chan<- cleanupRecord
	errRMu              sync.RWMutex
	sMuMu               sync.Mutex
	pcMuMu              sync.Mutex
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
		return l.getSingleton(ctx, record, serviceName)
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

func (l *locator) get(ctx context.Context, record record) (service any, cleanup Cleanup, err error) {
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

func (l *locator) getSingleton(ctx context.Context, record record, serviceName string) (any, error) {
	l.sMuMu.Lock()

	if _, ok := l.sMu[serviceName]; !ok {
		l.sMu[serviceName] = new(sync.Mutex)
	}

	mu := l.sMu[serviceName]

	l.sMuMu.Unlock()

	mu.Lock()
	defer mu.Unlock()

	service, ok := l.singletons.get(serviceName)
	if ok {
		return service, nil
	}

	service, cleanUp, err := l.get(ctx, record)
	if err != nil {
		return nil, err
	}

	go func() { l.singletonsCleanupCh <- cleanUp }()

	l.singletons.set(serviceName, service)

	return service, nil
}

func (l *locator) getPerContext(ctx context.Context, record record, serviceName string) (any, error) {
	if ctx == nil {
		return nil, newServiceBuilderError(ErrNilContext, record.lifetime, serviceName)
	}

	if err := ctx.Err(); err != nil {
		return nil, newServiceBuilderError(err, record.lifetime, serviceName)
	}

	l.pcMuMu.Lock()
	perContextKey := getPerContextKey(ctx, "")

	if _, ok := l.pcMu[perContextKey]; !ok {
		go func() {
			l.perContextCleanUpCh <- cleanupRecord{
				ctx: ctx,
				fn: func() {
					l.pcMuMu.Lock()

					delete(l.pcMu, perContextKey)
					l.perContext.delete(ctx)

					l.pcMuMu.Unlock()
				},
			}
		}()
	}

	perContextServiceKey := getPerContextKey(ctx, serviceName)
	if _, ok := l.pcMu[perContextServiceKey]; !ok {
		l.pcMu[perContextServiceKey] = new(sync.Mutex)
	}

	mu := l.pcMu[perContextServiceKey]

	l.pcMuMu.Unlock()

	mu.Lock()
	defer mu.Unlock()

	service, ok := l.perContext.get(ctx, serviceName)
	if ok {
		return service, nil
	}

	service, cleanUp, err := l.get(ctx, record)
	if err != nil {
		return nil, err
	}

	go func() { l.perContextCleanUpCh <- cleanupRecord{ctx: ctx, fn: cleanUp} }()

	l.perContext.set(ctx, serviceName, service)

	return service, nil
}
