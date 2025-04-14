package gc

import (
	. "experiments/benchmarks/metrics"
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

func (n *Node) run(req *request) {
	allocationStart := time.Now()
	req = new(request)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for *req = range n.reqs {
		switch req.op {
		case OpInsert:
			if req.value < n.value {
				if n.left == nil {
					Latency.Add(time.Since(req.latencyStart).Nanoseconds())

					allocationStart = time.Now()
					n.left = new(Node)
					n.left.reqs = make(chan request)
					n.left.done = make(chan bool)
					AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

					n.left.value = req.value

					// To avoid escape analysis
					r := request{}
					go n.left.run(&r)

					req.result <- true
				} else {
					n.left.reqs <- *req
				}
			} else if req.value > n.value {
				if n.right == nil {
					Latency.Add(time.Since(req.latencyStart).Nanoseconds())

					allocationStart = time.Now()
					n.right = new(Node)
					n.right.reqs = make(chan request)
					n.right.done = make(chan bool)
					AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

					n.right.value = req.value

					// To avoid escape analysis
					r := request{}
					go n.right.run(&r)

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

func NewFineGrainBinaryTree() *FineGrainBinaryTree {
	allocationStart := time.Now()
	t := new(FineGrainBinaryTree)
	t.reqs = make(chan request)
	t.done = make(chan bool)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	req := request{}
	go t.run(&req)

	return t
}

func (t *FineGrainBinaryTree) run(req *request) {
	allocationStart := time.Now()
	req = new(request)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for *req = range t.reqs {
		switch req.op {
		case OpInsert:
			if t.root == nil {
				Latency.Add(time.Since(req.latencyStart).Nanoseconds())

				allocationStart = time.Now()
				t.root = new(Node)
				t.root.reqs = make(chan request)
				t.root.done = make(chan bool)
				AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

				t.root.value = req.value

				r := request{}
				go t.root.run(&r)

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

func generateBinaryTreeOperations(valueRange int, op int, tree *FineGrainBinaryTree, done chan bool, req *request, i *int) {
	allocationStart := time.Now()
	req = new(request)
	req.result = make(chan bool)
	AllocationTime.Add(time.Since(allocationStart).Nanoseconds())

	for i = new(int); *i < op; *i++ {
		tree.Insert(*i + valueRange, *req)
		/*val := rand.IntN(valueRange) + 1
		switch method := opType(rand.IntN(2)); method {
		case OpInsert:
			tree.Insert(val, *req)
		case OpSearch:
			tree.Search(val, *req)
		}*/
	}
	done <- true
}

func (n *Node) destroyTree() {
	close(n.reqs)
	<-n.done
	if n.right != nil {
		n.right.destroyTree()
	}
	if n.left != nil {
		n.left.destroyTree()
	}
}

func RunBinaryTree(op int) SystemMetrics {
	debug.SetGCPercent(-1)

	ComputationTime.Store(0)
	AllocationTime.Store(0)
	DeallocationTime.Store(0)
	Latency.Store(0)

	// To avoid escape analysis
	reqs := [Goroutines]request{}
	c := [Goroutines]int{}

	computationTimeStart := time.Now()

	allocationTimeStart := time.Now()
	done := make(chan bool)
	AllocationTime.Add(time.Since(allocationTimeStart).Nanoseconds())

	fgbt := NewFineGrainBinaryTree()

	valueRange := 0
	for i := 0; i < Goroutines; i++ {
		go generateBinaryTreeOperations(valueRange, op, fgbt, done, &reqs[i], &c[i])
		valueRange += BinOp
	}

	for i := 0; i < Goroutines; i++ {
		<-done
	}

	// Decrement each reference counter
	fgbt.root.destroyTree()
	close(fgbt.reqs)
	<-fgbt.done

	deallocationStart := time.Now()
	runtime.GC()
	DeallocationTime.Add(time.Since(deallocationStart).Nanoseconds())

	ComputationTime.Store(time.Since(computationTimeStart).Nanoseconds())

	return SystemMetrics{
		float64(ComputationTime.Load()),
		float64(BinOp*Goroutines) / float64(ComputationTime.Load()),
		float64(Latency.Load()),
		float64(AllocationTime.Load()),
		float64(DeallocationTime.Load())}
}
