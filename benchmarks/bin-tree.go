package benchmarks

import (
	"fmt"
	"math/rand/v2"
	"time"
)

type OpType int

const (
	OpInsert OpType = iota
	OpRemove
	OpSearch
)

type Request struct {
	Op     OpType
	Value  int
	Result chan bool
	Time   time.Time
}

type Metrics struct {
	computationTime time.Duration
	latency         time.Duration
	operation       OpType
}

type ConcurrentBinTree struct {
	requests chan Request
	metrics  chan Metrics
}

func NewConcurrentBinTree(sz int) *ConcurrentBinTree {
	cbt := &ConcurrentBinTree{
		requests: make(chan Request),
		metrics:  make(chan Metrics, sz),
	}
	go cbt.run()
	return cbt
}

func (cbt *ConcurrentBinTree) Insert(val int, requestTime time.Time) {
	cbt.requests <- Request{Op: OpInsert, Value: val, Time: requestTime}
}

func (cbt *ConcurrentBinTree) Remove(val int, requestTime time.Time) {
	cbt.requests <- Request{Op: OpRemove, Value: val, Time: requestTime}
}

func (cbt *ConcurrentBinTree) Search(val int, requestTime time.Time) bool {
	response := make(chan bool)
	cbt.requests <- Request{Op: OpSearch, Value: val, Result: response, Time: requestTime}
	return <-response
}

func (cbt *ConcurrentBinTree) run() {
	tree := BinTree{}
	for req := range cbt.requests {
		var tl time.Duration
		var tc time.Duration
		switch req.Op {
		case OpInsert:
			tl = time.Since(req.Time)
			startTime := time.Now()
			tree = *tree.Insert(req.Value)
			tc = time.Since(startTime)
		case OpRemove:
			tl = time.Since(req.Time)
			startTime := time.Now()
			tree = *tree.Remove(req.Value)
			tc = time.Since(startTime)
		case OpSearch:
			tl = time.Since(req.Time)
			startTime := time.Now()
			req.Result <- tree.Search(req.Value)
			tc = time.Since(startTime)
		}
		cbt.metrics <- Metrics{operation: req.Op, latency: tl, computationTime: tc}
	}
}

type BinTree struct {
	Left  *BinTree
	Val   int
	Right *BinTree
}

func (tree *BinTree) Insert(val int) *BinTree {
	if tree == nil || tree.Val == 0 {
		return &BinTree{Val: val}
	}

	// If value is already present
	if tree.Val == val {
		return tree
	}

	if tree.Val < val {
		tree.Right = tree.Right.Insert(val)
	} else {
		tree.Left = tree.Left.Insert(val)
	}
	return tree
}

func (tree *BinTree) Search(val int) bool {
	if tree == nil || tree.Val == 0 {
		return false
	}
	if tree.Val < val {
		return tree.Right.Search(val)
	} else if tree.Val > val {
		return tree.Left.Search(val)
	} else {
		return true
	}
}

func (tree *BinTree) Remove(val int) *BinTree {
	if tree == nil {
		return tree
	}
	if tree.Val < val {
		tree.Right = tree.Right.Remove(val)
	} else if tree.Val > val {
		tree.Left = tree.Left.Remove(val)
	} else {
		if tree.Left == nil && tree.Right == nil {
			tree.Val = 0
			return tree
		}
		// When tree has 0 children
		if tree.Left == nil {
			return tree.Right
		}
		// When root has only 1 left child
		if tree.Right == nil {
			return tree.Left
		}

		// When both are present
		succ := tree.Right.FindSuccessor()
		tree.Val = succ.Val
		tree.Right = tree.Right.Remove(succ.Val)
	}
	return tree
}

func (tree *BinTree) FindSuccessor() *BinTree {
	if tree.Left == nil {
		return tree
	}
	return tree.Left.FindSuccessor()
}

func (tree *BinTree) Print() {
	if tree == nil {
		return
	}
	fmt.Print(tree.Val, " ")
	if tree.Left != nil {
		fmt.Print("Left of ", tree.Val, ": ")
		tree.Left.Print()
	}

	if tree.Right != nil {
		fmt.Print("Right of ", tree.Val, ": ")
		tree.Right.Print()
	}
}

func ConcurrentBinaryTree(n int, valueRange int, op int) ([3]time.Duration, [3]time.Duration, [3]float64, [3]int) {
	cbt := NewConcurrentBinTree(op * n)

	for i := 0; i < n; i++ {
		go func() {
			for range op {
				method := OpType(rand.IntN(3))
				val := rand.IntN(valueRange) + 1
				switch method {
				case OpInsert:
					cbt.Insert(val, time.Now())
				case OpRemove:
					cbt.Remove(val, time.Now())
				case OpSearch:
					cbt.Search(val, time.Now())
				}
			}
		}()
	}

	var totalLatency [3]time.Duration
	var totalComputationTime [3]time.Duration
	var totalOperations [3]int

	for i := 0; i < op*n; i++ {
		res := <-cbt.metrics
		totalLatency[res.operation] += res.latency
		totalComputationTime[res.operation] += res.computationTime
		totalOperations[res.operation]++
	}

	var avgLatency [3]time.Duration
	var avgComputationTime [3]time.Duration
	var totalThroughput [3]float64

	for i := 0; i < 3; i++ {
		if totalOperations[i] != 0 {
			avgLatency[i] = totalLatency[i] / time.Duration(totalOperations[i])
			avgComputationTime[i] = totalComputationTime[i] / time.Duration(totalOperations[i])
			totalThroughput[i] = float64(totalOperations[i]) / float64(totalComputationTime[i].Seconds())
		}
	}

	return avgLatency, avgComputationTime, totalThroughput, totalOperations
}
