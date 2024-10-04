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
	"slices"
	"sync"
	"syscall"
	"time"
)

var reflectValuesPool = sync.Pool{
	New: func() any {
		return make([]reflect.Value, 0, 1)
	},
}

var _ ServiceLocator = new(locator)

type cleanupRecord struct {
	ctx context.Context
	cleanupNodeUpdate
}

type cleanupNodeUpdate struct {
	fn Cleanup
	id uintptr
}

type cleanupNode interface {
	clean()
	updateCleanupNode(cleanupNodeUpdate)
	setId(uintptr)
}

type cleanupNodeImpl struct {
	fn         Cleanup
	dependants []*cleanupNodeImpl
	id         uintptr
	cleaned    bool
}

func (ct *cleanupNodeImpl) clean() {
	for _, nodes := range ct.dependants {
		nodes.clean()
	}

	if !ct.cleaned {
		ct.fn()
	}

	ct.cleaned = true
}

func (node *cleanupNodeImpl) updateCleanupNode(update cleanupNodeUpdate) {
	if node.id == update.id {
		node.fn = update.fn
		return
	}

	for _, n := range node.dependants {
		n.updateCleanupNode(update)
	}
}

func (node *cleanupNodeImpl) setId(id uintptr) {
	node.id = id
}

type singleCleanupFn func()

func (fn singleCleanupFn) clean() {
	fn()
}

func (fn singleCleanupFn) updateCleanupNode(update cleanupNodeUpdate) {
	fn = singleCleanupFn(update.fn)
}

func (fn singleCleanupFn) setId(id uintptr) {
}

type cleanupNodeRecord struct {
	*cleanupNodeImpl
	typeName     string
	dependencies []string
}

func buildCleanupNodes(records []*record) cleanupNode {
	hasNoDeps := true
	for _, rec := range records {
		if rec.constructorType == withErrorAndCleanUp {
			hasNoDeps = false
		}
	}

	if hasNoDeps {
		return singleCleanupFn(func() {})
	}

	headNode := &cleanupNodeImpl{fn: func() {}}

	nodes := make([]*cleanupNodeRecord, 0)
	for _, rec := range records {
		nodes = append(nodes, buildCleanupNodeRecord(rec, records))
	}

	for _, node := range nodes {
		buildCleanupNodeRecordDependants(node, nodes)
	}

	headNode.dependants = filterOnlyTopNodes(nodes)

	return headNode
}

func filterOnlyTopNodes(nodes []*cleanupNodeRecord) []*cleanupNodeImpl {
	result := make([]*cleanupNodeImpl, 0)

	for _, n := range nodes {
		if len(n.dependencies) == 0 {
			result = append(result, n.cleanupNodeImpl)
		}
	}

	return result
}

func buildCleanupNodeRecordDependants(node *cleanupNodeRecord, nodes []*cleanupNodeRecord) {
	for _, n := range nodes {
		if slices.Contains(n.dependencies, node.typeName) {
			node.dependants = append(node.dependants, n.cleanupNodeImpl)
		}
	}
}

func buildCleanupNodeRecord(rec *record, records []*record) *cleanupNodeRecord {
	node := &cleanupNodeImpl{
		fn: func() {},
		id: rec.id,
	}

	deps := make([]string, 0)
	for _, depName := range rec.dependencies {
		for _, dep := range records {
			if rec.constructorType == withErrorAndCleanUp && dep.typeName == depName && dep.lifetime == rec.lifetime {
				deps = append(deps, depName)
			}
		}
	}

	nodeRec := &cleanupNodeRecord{
		typeName:        rec.typeName,
		dependencies:    deps,
		cleanupNodeImpl: node,
	}

	return nodeRec
}

// worker to handle singletons cleanup before application exit
func singletonCleanupWorker(
	ctx context.Context, cancel context.CancelFunc, cleanupSchema cleanupNode,
	singletonsCleanupCh <-chan cleanupNodeUpdate, wg *sync.WaitGroup,
) {
	var cleanup Cleanup = func() { cleanupSchema.clean() }

loop:
	for {
		select {
		case fn := <-singletonsCleanupCh:
			cleanupSchema.updateCleanupNode(fn)
		case <-ctx.Done():
			cleanup.CallWithRecovery(Singleton)
			break loop
		}
	}

	wg.Wait()
	cancel()
}

// worker to handle per-context cleanups
func perContextCleanupWorker(
	ctx context.Context,
	getCleanupNode func(context.Context) cleanupNode,
	perContextCleanupCh <-chan cleanupRecord,
	wg *sync.WaitGroup,
) {
	cleanups := make(map[uintptr]cleanupNode)
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
			node, ok := cleanups[pt]

			if !ok {
				node = getCleanupNode(rec.ctx)
				cleanups[pt] = node
			}

			node.updateCleanupNode(rec.cleanupNodeUpdate)

			if replaceNextContext {
				nextCtx = rec.ctx
				replaceNextContext = false
			}

			ctxList = append(ctxList, rec.ctx)
		case <-nextCtx.Done():
			pt := reflect.ValueOf(nextCtx).Pointer()
			if node, ok := cleanups[pt]; ok {
				var fn Cleanup = func() { node.clean() }
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

				j := rand.IntN(len(ctxList))
				ctxList[0], ctxList[j] = ctxList[j], ctxList[0]

				nextCtx = ctxList[0]
			}
		case <-ctx.Done():
			break loop
		}
	}

	ticker.Stop()

	for _, node := range cleanups {
		var fn Cleanup = func() { node.clean() }
		fn.CallWithRecovery(PerContext)
	}

	wg.Done()
}

func newLocator(ctx context.Context, constructors map[string]*record) ServiceLocator {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	singletonsCleanupCh := make(chan cleanupNodeUpdate)
	perContextCleanupCh := make(chan cleanupRecord)
	var wg sync.WaitGroup

	singletons := make([]*record, 0)
	perContexts := make([]*record, 0)

	perCtxIDs := make([]uintptr, 0)
	for _, rec := range constructors {
		switch rec.lifetime {
		case Singleton:
			singletons = append(singletons, rec)
		case PerContext:
			perCtxIDs = append(perCtxIDs, rec.id)
			perContexts = append(perContexts, rec)
		}
	}
	go singletonCleanupWorker(ctx, cancel, buildCleanupNodes(singletons), singletonsCleanupCh, &wg)

	wg.Add(1)
	go perContextCleanupWorker(
		ctx,
		func(ctx context.Context) cleanupNode {
			n := buildCleanupNodes(perContexts)
			n.setId(reflect.ValueOf(ctx).Pointer())

			return n
		},
		perContextCleanupCh,
		&wg,
	)

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
	singletonsCleanupCh chan<- cleanupNodeUpdate
	perContextCleanUpCh chan<- cleanupRecord
	sMu                 sync.Map
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
	args := reflectValuesPool.Get().([]reflect.Value)
	defer func() {
		args = args[:0]
		reflectValuesPool.Put(args)
	}()

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

	if record.constructorType == withErrorAndCleanUp {
		go func() {
			l.singletonsCleanupCh <- cleanupNodeUpdate{
				id: record.id,
				fn: func() {
					cleanUp()
					l.sMu.Delete(record.id)
				},
			}
		}()
	}

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

	switch {
	case !ok && record.constructorType == withErrorAndCleanUp:
		go func() {
			l.perContextCleanUpCh <- cleanupRecord{
				ctx: ctx,
				cleanupNodeUpdate: cleanupNodeUpdate{
					id: record.id,
					fn: func() { cleanUp() },
				},
			}
			l.perContextCleanUpCh <- cleanupRecord{
				ctx: ctx,
				cleanupNodeUpdate: cleanupNodeUpdate{
					id: ctxKey,
					fn: func() { l.perContext.delete(ctxKey) },
				},
			}
		}()
	case !ok:
		go func() {
			l.perContextCleanUpCh <- cleanupRecord{
				ctx: ctx,
				cleanupNodeUpdate: cleanupNodeUpdate{
					id: record.id,
					fn: func() { l.perContext.delete(ctxKey) },
				},
			}
		}()
	case record.constructorType == withErrorAndCleanUp:
		go func() {
			l.perContextCleanUpCh <- cleanupRecord{
				ctx: ctx,
				cleanupNodeUpdate: cleanupNodeUpdate{
					id: record.id,
					fn: cleanUp,
				},
			}
		}()
	}

	scope.value = &service

	return service, nil
}
