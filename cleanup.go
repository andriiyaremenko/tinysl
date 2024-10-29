package tinysl

import (
	"context"
	"slices"
)

var _ ServiceLocator = new(locator)

type cleanupNodeUpdate struct {
	fn Cleanup
	id int
}

type cleanupNode interface {
	len() int
	clean()
	zeroOut()
	updateCleanupNode(int, Cleanup)
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

func (ct *cleanupNodeImpl) len() int {
	return len(ct.dependants)
}

func (ct *cleanupNodeImpl) zeroOut() {
	ct.cleaned = false
	ct.fn = func() {}
	for _, nodes := range ct.dependants {
		nodes.zeroOut()
	}
}

func (node *cleanupNodeImpl) updateCleanupNode(id int, fn Cleanup) {
	if node.id == id {
		node.fn = fn
		return
	}

	for _, n := range node.dependants {
		n.updateCleanupNode(id, fn)
	}
}

type singleCleanupFn func()

func (fn singleCleanupFn) len() int {
	return 0
}

func (fn singleCleanupFn) clean() {
	fn()
}

func (fn singleCleanupFn) zeroOut() {
}

func (fn singleCleanupFn) updateCleanupNode(int, Cleanup) {
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
		return &cleanupNodeImpl{fn: func() {}}
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
	singletonsCleanupCh <-chan cleanupNodeUpdate,
) {
	var cleanup Cleanup = func() { cleanupSchema.clean() }

loop:
	for {
		select {
		case update := <-singletonsCleanupCh:
			cleanupSchema.updateCleanupNode(update.id, update.fn)
		case <-ctx.Done():
			cleanup.CallWithRecovery(Singleton)
			break loop
		}
	}

	cancel()
}
