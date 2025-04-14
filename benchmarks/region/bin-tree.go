//go:build goexperiment.regions

package region

import (
	. "experiments/benchmarks/metrics"
	"fmt"
	"region"
	"runtime"
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
	done  chan bool
}

type FineGrainBinaryTree struct {
	root *Node
	reqs chan request
	done chan bool
}

func (n *Node) run(r1 *region.Region) {
	allocationStart := time.Now()
	req := region.AllocFromRegion[request](r1)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for *req = range n.reqs {
		switch req.op {
		case OpInsert:
			if req.value < n.value {
				if n.left == nil {
					Latency.Add(time.Since(req.latencyStart).Nanoseconds())

					allocationStart = time.Now()
					n.left = region.AllocFromRegion[Node](r1)
					n.left.reqs = region.AllocChannel[request](0, r1)
					n.left.done = region.AllocChannel[bool](0, r1)
					AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

					n.left.value = req.value

					if r1.IncRefCounter() {
						go n.left.run(r1)
					}

					req.result <- true
				} else {
					n.left.reqs <- *req
				}
			} else if req.value > n.value {
				if n.right == nil {
					Latency.Add(time.Since(req.latencyStart).Nanoseconds())

					allocationStart = time.Now()
					n.right = region.AllocFromRegion[Node](r1)
					n.right.reqs = region.AllocChannel[request](0, r1)
					n.right.done = region.AllocChannel[bool](0, r1)
					AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

					n.right.value = req.value

					if r1.IncRefCounter() {
						go n.right.run(r1)
					}

					req.result <- true
				} else {
					n.right.reqs <- *req
				}
			} else {
				Latency.Add(time.Since(req.latencyStart).Nanoseconds())
				req.result <- false
			}
		case OpSearch:
			if req.value == n.value {
				Latency.Add(time.Since(req.latencyStart).Nanoseconds())
				req.result <- true
			} else if req.value < n.value && n.left != nil {
				n.left.reqs <- *req
			} else if req.value > n.value && n.right != nil {
				n.right.reqs <- *req
			} else {
				Latency.Add(time.Since(req.latencyStart).Nanoseconds())
				req.result <- false
			}
		}
	}
	n.done <- true
}

func NewFineGrainBinaryTree(r *region.Region) *FineGrainBinaryTree {
	allocationStart := time.Now()
	t := region.AllocFromRegion[FineGrainBinaryTree](r)
	t.reqs = region.AllocChannel[request](0, r)
	t.done = region.AllocChannel[bool](0, r)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	if r.IncRefCounter() {
		go t.run(r)
	}

	return t
}

func (t *FineGrainBinaryTree) run(r1 *region.Region) {
	allocationStart := time.Now()
	req := region.AllocFromRegion[request](r1)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for *req = range t.reqs {
		switch req.op {
		case OpInsert:
			if t.root == nil {
				Latency.Add(time.Since(req.latencyStart).Nanoseconds())

				allocationStart = time.Now()
				t.root = region.AllocFromRegion[Node](r1)
				t.root.reqs = region.AllocChannel[request](0, r1)
				t.root.done = region.AllocChannel[bool](0, r1)
				AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

				t.root.value = req.value

				if r1.IncRefCounter() {
					go t.root.run(r1)
				}

				req.result <- true
			} else {
				t.root.reqs <- *req
			}
		case OpSearch:
			if t.root == nil {
				Latency.Add(time.Since(req.latencyStart).Nanoseconds())
				req.result <- false
			} else {
				t.root.reqs <- *req
			}
		}
	}
	t.done <- true
}

func (tree *FineGrainBinaryTree) Insert(value int, req request) {
	req.value = value
	req.op = OpInsert
	req.latencyStart = time.Now()

	tree.reqs <- req
	<-req.result
}

func (tree *FineGrainBinaryTree) Search(value int, req request) {
	req.value = value
	req.op = OpSearch
	req.latencyStart = time.Now()

	tree.reqs <- req
	<-req.result
}

func generateBinaryTreeOperations(valueRange int, op int, tree *FineGrainBinaryTree, done chan bool, r1 *region.Region) {
	r2 := region.CreateRegion(0)

	allocationStart := time.Now()
	req := region.AllocFromRegion[request](r2)
	req.result = region.AllocChannel[bool](0, r2)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for i := region.AllocFromRegion[int](r2); *i < op; *i++ {
		tree.Insert(*i + valueRange, *req)
		/*val := rand.IntN(valueRange) + 1
		switch method := opType(rand.IntN(2)); method {
		case OpInsert:
			tree.Insert(val, *req)
		case OpSearch:
			tree.Search(val, *req)
		}*/
	}
	deallocationStart := time.Now()
	r2.RemoveRegion()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

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

func (n *Node) destroyTree(r *region.Region) {
	close(n.reqs)
	<-n.done
	r.DecRefCounter()
	if n.right != nil {
		n.right.destroyTree(r)
	}
	if n.left != nil {
		n.left.destroyTree(r)
	}
}

func RunBinaryTree(op int) SystemMetrics {
	debug.SetGCPercent(-1)

	ComputationTime.Store(0)
	AllocationTime.Store(0)
	DeallocationTime.Store(0)
	Latency.Store(0)

	computationTimeStart := time.Now()
	//r1 := region.CreateRegion(BinRange * 350)
	r1 := region.CreateRegion(RegionBlockBytes / 8)

	allocationTimeStart := time.Now()
	done := region.AllocChannel[bool](0, r1)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	fgbt := NewFineGrainBinaryTree(r1)

	valueRange := 0
	for i := 0; i < Goroutines; i++ {
		if r1.IncRefCounter() {
			go generateBinaryTreeOperations(valueRange, op, fgbt, done, r1)
		}
		valueRange += BinOp
	}

	for i := 0; i < Goroutines; i++ {
		<-done
	}

	// Decrement each reference counter
	fgbt.root.destroyTree(r1)
	close(fgbt.reqs)
	<-fgbt.done
	r1.DecRefCounter()

	deallocationStart := time.Now()
	r1.RemoveRegion()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	ComputationTime.Store(time.Since(computationTimeStart).Nanoseconds())

	runtime.GC()

	return SystemMetrics{
		float64(ComputationTime.Load()),
		float64(BinOp*Goroutines) / float64(ComputationTime.Load()),
		float64(Latency.Load()),
		float64(AllocationTime.Load()),
		float64(DeallocationTime.Load())}
}
