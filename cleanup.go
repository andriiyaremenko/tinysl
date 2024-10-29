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

type cleanupNode struct {
	fn         Cleanup
	dependants []*cleanupNode
	id         int
}

func (ct *cleanupNode) clean() {
	for _, nodes := range ct.dependants {
		nodes.clean()
	}

	ct.fn()

	ct.fn = func() {}
}

func (ct *cleanupNode) empty() bool {
	return len(ct.dependants) == 0
}

func (node *cleanupNode) updateCleanupNode(id int, fn Cleanup) {
	if node.id == id {
		node.fn = fn
		return
	}

	for _, n := range node.dependants {
		n.updateCleanupNode(id, fn)
	}
}

type cleanupNodeRecord struct {
	*cleanupNode
	typeName     string
	dependencies []int
}

func buildCleanupNodes(records []*locatorRecord) *cleanupNode {
	hasNoDeps := true
	for _, rec := range records {
		if rec.constructorType == withErrorAndCleanUp {
			hasNoDeps = false
		}
	}

	if hasNoDeps {
		return &cleanupNode{fn: func() {}}
	}

	headNode := &cleanupNode{fn: func() {}}

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

func filterOnlyTopNodes(nodes []*cleanupNodeRecord) []*cleanupNode {
	result := make([]*cleanupNode, 0)

	for _, n := range nodes {
		if len(n.dependencies) == 0 {
			result = append(result, n.cleanupNode)
		}
	}

	return result
}

func buildCleanupNodeRecordDependants(node *cleanupNodeRecord, nodes []*cleanupNodeRecord) {
	for _, n := range nodes {
		if slices.Contains(n.dependencies, node.id) {
			node.dependants = append(node.dependants, n.cleanupNode)
		}
	}
}

func buildCleanupNodeRecord(rec *locatorRecord, records []*locatorRecord) *cleanupNodeRecord {
	node := &cleanupNode{
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
		typeName:     rec.typeName,
		dependencies: deps,
		cleanupNode:  node,
	}

	return nodeRec
}

// worker to handle singletons cleanup before application exit
func singletonCleanupWorker(
	ctx context.Context, cancel context.CancelFunc, cleanupSchema *cleanupNode,
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
