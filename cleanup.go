package tinysl

import (
	"context"
	"math/rand/v2"
	"reflect"
	"slices"
	"sync"
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
	id int
}

type cleanupNode interface {
	clean()
	zeroOut()
	updateCleanupNode(cleanupNodeUpdate)
}

type cleanupNodeImpl struct {
	fn         Cleanup
	dependants []*cleanupNodeImpl
	id         int
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

func (ct *cleanupNodeImpl) zeroOut() {
	ct.cleaned = false
	ct.fn = func() {}
	for _, nodes := range ct.dependants {
		nodes.zeroOut()
	}
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

type singleCleanupFn func()

func (fn singleCleanupFn) clean() {
	fn()
}

func (fn singleCleanupFn) zeroOut() {
	fn = func() {}
}

func (fn singleCleanupFn) updateCleanupNode(update cleanupNodeUpdate) {
	fn = singleCleanupFn(update.fn)
}

type cleanupNodeRecord struct {
	*cleanupNodeImpl
	typeName     string
	dependencies []int
}

func buildCleanupNodes(records []*locatorRecord) cleanupNode {
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
		if slices.Contains(n.dependencies, node.id) {
			node.dependants = append(node.dependants, n.cleanupNodeImpl)
		}
	}
}

func buildCleanupNodeRecord(rec *locatorRecord, records []*locatorRecord) *cleanupNodeRecord {
	node := &cleanupNodeImpl{
		fn: func() {},
		id: rec.id,
	}

	deps := make([]int, 0)
	for _, depRecord := range rec.dependencies {
		for _, dep := range records {
			if rec.constructorType == withErrorAndCleanUp && dep.id == depRecord.id && dep.lifetime == rec.lifetime {
				deps = append(deps, depRecord.id)
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
	wg *sync.WaitGroup,
	perContextCleanupCh <-chan cleanupRecord,
	getCleanupNode func() cleanupNode,
) {
	pool := sync.Pool{
		New: func() any {
			return getCleanupNode()
		},
	}

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
				node = pool.Get().(cleanupNode)
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
				var fn Cleanup = node.clean
				fn.CallWithRecovery(PerContext)

				node.zeroOut()
				pool.Put(node)
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
		var fn Cleanup = node.clean
		fn.CallWithRecovery(PerContext)
	}

	wg.Done()
}
