//go:build goexperiment.regions

package region

import (
	. "experiments/benchmarks/metrics"
	"fmt"
	"math/rand/v2"
	"region"
	"runtime/debug"
	"time"
)

type opType int

const (
	OpInsert opType = iota
	OpSearch
	OpRemove
)

type request struct {
	value        int
	op           opType
	result       chan bool
	latencyStart time.Time
}

type Node struct {
	value int
	left  *Node
	right *Node
	reqs  chan request
}

type FineGrainBinaryTree struct {
	root *Node
	reqs chan request
}

func (n *Node) run(r *region.Region) {
	allocationStart := time.Now()
	req := region.AllocFromRegion[request](r)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for *req = range n.reqs {
		Latency.Add(time.Since(req.latencyStart).Nanoseconds())

		switch req.op {
		case OpInsert:
			if req.value < n.value {
				if n.left == nil {

					allocationStart = time.Now()
					n.left = region.AllocFromRegion[Node](r)
					n.left.reqs = region.AllocChannel[request](0, r)
					AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

					n.left.value = req.value

					if r.IncRefCounter() {
						go n.left.run(r)
					}

					req.result <- true
				} else {
					req.latencyStart = time.Now()
					n.left.reqs <- *req
				}
			} else if req.value > n.value {
				if n.right == nil {

					allocationStart = time.Now()
					n.right = region.AllocFromRegion[Node](r)
					n.right.reqs = region.AllocChannel[request](0, r)
					AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

					n.right.value = req.value

					if r.IncRefCounter() {
						go n.right.run(r)
					}

					req.result <- true
				} else {
					req.latencyStart = time.Now()
					n.right.reqs <- *req
				}
			} else {
				req.result <- false
			}
		case OpSearch:
			if req.value == n.value {
				req.result <- true
			} else if req.value < n.value && n.left != nil {
				req.latencyStart = time.Now()
				n.left.reqs <- *req
			} else if req.value > n.value && n.right != nil {
				req.latencyStart = time.Now()
				n.right.reqs <- *req
			} else {
				req.result <- false
			}
		}
	}
	r.DecRefCounter()
}

func NewFineGrainBinaryTree(r *region.Region) *FineGrainBinaryTree {
	allocationStart := time.Now()
	t := region.AllocFromRegion[FineGrainBinaryTree](r)
	t.reqs = region.AllocChannel[request](0, r)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	if r.IncRefCounter() {
		go t.run(r)
	}

	return t
}

func (t *FineGrainBinaryTree) run(r *region.Region) {
	allocationStart := time.Now()
	req := region.AllocFromRegion[request](r)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for *req = range t.reqs {
		Latency.Add(time.Since(req.latencyStart).Nanoseconds())
		switch req.op {
		case OpInsert:
			if t.root == nil {
				allocationStart = time.Now()
				t.root = region.AllocFromRegion[Node](r)
				t.root.reqs = region.AllocChannel[request](0, r)
				AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

				t.root.value = req.value

				if r.IncRefCounter() {
					go t.root.run(r)
				}

				req.result <- true
			} else {
				req.latencyStart = time.Now()
				t.root.reqs <- *req
			}
		case OpSearch:
			if t.root == nil {
				req.result <- false
			} else {
				req.latencyStart = time.Now()
				t.root.reqs <- *req
			}
		}
	}
	r.DecRefCounter()
}

func (tree *FineGrainBinaryTree) Insert(value int, r *region.Region) bool {
	allocationStart := time.Now()
	req := region.AllocFromRegion[request](r)
	req.result = region.AllocChannel[bool](0, r)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	req.value = value
	req.op = OpInsert
	req.latencyStart = time.Now()

	tree.reqs <- *req
	return <-req.result
}

func (tree *FineGrainBinaryTree) Search(value int, r *region.Region) bool {
	allocationStart := time.Now()
	req := region.AllocFromRegion[request](r)
	req.result = region.AllocChannel[bool](0, r)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	req.value = value
	req.op = OpSearch
	req.latencyStart = time.Now()

	tree.reqs <- *req
	return <-req.result
}

func generateBinaryTreeOperations(valueRange int, op int, tree *FineGrainBinaryTree, done chan bool, r1 *region.Region) {
	for i := region.AllocFromRegion[int](r1); *i < op; *i++ {
		val := rand.IntN(valueRange) + 1
		switch method := opType(rand.IntN(2)); method {
		case OpInsert:
			tree.Insert(val, r1)
		case OpSearch:
			tree.Search(val, r1)
		}
	}
	r1.DecRefCounter()
	done <- true
}

func (n *Node) Print() {
	fmt.Print(n.value)
	if n.right != nil {
		fmt.Print(", ", n.value, " Right ")
		n.right.Print()
	}
	if n.left != nil {
		fmt.Print(", ", n.value, " Left ")
		n.left.Print()
	}
}

func RunBinaryTree(valueRange int, op int) Metrics {
	debug.SetGCPercent(-1)

	ComputationTime.Store(0)
	AllocationTime.Store(0)
	DeallocationTime.Store(0)
	Latency.Store(0)

	computationTimeStart := time.Now()
	r1 := region.CreateRegion()
	done := region.AllocChannel[bool](0, r1)

	fgbt := NewFineGrainBinaryTree(r1)

	for i := 0; i < Goroutines; i++ {
		if r1.IncRefCounter() {
			go generateBinaryTreeOperations(valueRange, op, fgbt, done, r1)
		}
	}

	for i := 0; i < Goroutines; i++ {
		<-done
	}

	ComputationTime.Store(time.Since(computationTimeStart).Nanoseconds())

	deallocationStart := time.Now()
	r1.RemoveRegion()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	return Metrics{
		float64(ComputationTime.Load()) / 1_000,
		float64(BinOp*Goroutines*1_000_000) / float64(ComputationTime.Load()),
		float64(Latency.Load()) / 1_000,
		float64(AllocationTime.Load()) / 1_000,
		float64(DeallocationTime.Load()) / 1_000}
}
