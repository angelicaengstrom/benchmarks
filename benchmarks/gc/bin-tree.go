package gc

import (
	. "experiments/benchmarks/metrics"
	"math/rand/v2"
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
	value  int
	op     opType
	result chan bool
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

func (n *Node) run() {
	allocationStart := time.Now()
	req := new(request)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	var memStats runtime.MemStats

	latencyStart := time.Now()
	for *req = range n.reqs {
		Latency.Add(time.Since(latencyStart).Nanoseconds())

		switch req.op {
		case OpInsert:
			if req.value < n.value {
				if n.left == nil {

					allocationStart = time.Now()
					n.left = new(Node)
					n.left.reqs = make(chan request)
					AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

					n.left.value = req.value

					go n.left.run()

					req.result <- true
				} else {
					n.left.reqs <- *req
				}
			} else if req.value > n.value {
				if n.right == nil {

					allocationStart = time.Now()
					n.right = new(Node)
					n.right.reqs = make(chan request)
					AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

					n.right.value = req.value

					go n.right.run()

					req.result <- true
				} else {
					n.right.reqs <- *req
				}
			} else {
				req.result <- false
			}
		case OpSearch:
			if req.value == n.value {
				req.result <- true
			} else if req.value < n.value && n.left != nil {
				n.left.reqs <- *req
			} else if req.value > n.value && n.right != nil {
				n.right.reqs <- *req
			} else {
				req.result <- false
			}
		}

		runtime.ReadMemStats(&memStats)

		if memStats.HeapAlloc > P_memoryConsuption.Load() {
			P_memoryConsuption.Store(memStats.HeapAlloc)
		}

		externalFrag := float64(memStats.HeapIdle) / float64(memStats.HeapSys)
		if externalFrag > P_externalFrag.Load().(float64) {
			P_externalFrag.Store(externalFrag)
		}

		internalFrag := float64(memStats.HeapIntFrag) / float64(memStats.HeapAlloc)
		if internalFrag > P_internalFrag.Load().(float64) {
			P_internalFrag.Store(internalFrag)
		}

		latencyStart = time.Now()
	}
}

func NewFineGrainBinaryTree() *FineGrainBinaryTree {
	allocationStart := time.Now()
	t := new(FineGrainBinaryTree)
	t.reqs = make(chan request)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	go t.run()

	return t
}

func (t *FineGrainBinaryTree) run() {
	allocationStart := time.Now()
	req := new(request)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	latencyStart := time.Now()
	for *req = range t.reqs {
		Latency.Add(time.Since(latencyStart).Nanoseconds())
		switch req.op {
		case OpInsert:
			if t.root == nil {
				allocationStart = time.Now()
				t.root = new(Node)
				t.root.reqs = make(chan request)
				AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

				t.root.value = req.value

				go t.root.run()

				req.result <- true
			} else {
				t.root.reqs <- *req
			}
		case OpSearch:
			if t.root == nil {
				req.result <- false
			} else {
				t.root.reqs <- *req
			}
		}
		latencyStart = time.Now()
	}
}

func (tree *FineGrainBinaryTree) Insert(value int) bool {
	allocationStart := time.Now()
	req := new(request)
	req.result = make(chan bool)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	req.value = value
	req.op = OpInsert

	tree.reqs <- *req
	return <-req.result
}

func (tree *FineGrainBinaryTree) Search(value int) bool {
	allocationStart := time.Now()
	req := new(request)
	req.result = make(chan bool)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	req.value = value
	req.op = OpSearch

	tree.reqs <- *req
	return <-req.result
}

func generateBinaryTreeOperations(valueRange int, op int, tree *FineGrainBinaryTree, done chan bool) {
	for i := new(int); *i < op; *i++ {
		val := rand.IntN(valueRange) + 1
		switch method := opType(rand.IntN(2)); method {
		case OpInsert:
			tree.Insert(val)
		case OpSearch:
			tree.Search(val)
		}
	}
	done <- true
}

func RunBinaryTree(valueRange int, op int) Metrics {
	debug.SetGCPercent(-1)

	ComputationTime.Store(0)
	AllocationTime.Store(0)
	DeallocationTime.Store(0)
	Latency.Store(0)
	P_memoryConsuption.Store(0)
	P_internalFrag.Store(0.0)
	P_externalFrag.Store(0.0)

	done := make(chan bool)

	computationTimeStart := time.Now()

	fgbt := NewFineGrainBinaryTree()

	for i := 0; i < Goroutines; i++ {
		go generateBinaryTreeOperations(valueRange, op, fgbt, done)
	}

	for i := 0; i < Goroutines; i++ {
		<-done
	}

	ComputationTime.Store(time.Since(computationTimeStart).Nanoseconds())

	deallocationStart := time.Now()
	runtime.GC()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	return Metrics{
		float64(ComputationTime.Load()) / 1_000_000_000,
		float64(BinOp*Goroutines*1_000_000_000) / float64(ComputationTime.Load()),
		float64(Latency.Load()) / 1_000_000_000,
		float64(P_memoryConsuption.Load()),
		P_externalFrag.Load().(float64),
		P_internalFrag.Load().(float64),
		float64(AllocationTime.Load()) / 1_000_000_000,
		float64(DeallocationTime.Load()) / 1_000_000_000}
}
