package benchmarks

import (
	"fmt"
	"math/rand/v2"
)

type opType int

const (
	OpInsert opType = iota
	OpRemove
	OpSearch
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
	for req := range n.reqs {
		switch req.op {
		case OpInsert:
			if req.value < n.value {
				if n.left == nil {
					n.left = &Node{value: req.value, reqs: make(chan request, 10)}
					go n.left.run()
				} else {
					n.left.reqs <- req
				}
			} else if req.value > n.value {
				if n.right == nil {
					n.right = &Node{value: req.value, reqs: make(chan request, 10)}
					go n.right.run()
				} else {
					n.right.reqs <- req
				}
			}
		case OpSearch:
			if req.value == n.value {
				req.result <- true
			} else if req.value < n.value && n.left != nil {
				n.left.reqs <- req
			} else if req.value > n.value && n.right != nil {
				n.right.reqs <- req
			} else {
				req.result <- false
			}
		}
	}
}

func NewFineGrainBinaryTree() *FineGrainBinaryTree {
	t := &FineGrainBinaryTree{reqs: make(chan request)}
	go t.run()
	return t
}

func (t *FineGrainBinaryTree) run() {
	var root *Node
	for req := range t.reqs {
		switch req.op {
		case OpInsert:
			if root == nil {
				root = &Node{value: req.value, reqs: make(chan request, 10)}
				go root.run()
			} else {
				root.reqs <- req
			}
		case OpSearch:
			if root == nil {
				req.result <- false
			} else {
				root.reqs <- req
			}
		}
	}
}

func (tree *FineGrainBinaryTree) Insert(value int) {
	tree.reqs <- request{value: value, op: OpInsert}
}

func (tree *FineGrainBinaryTree) Search(value int) bool {
	res := make(chan bool)
	tree.reqs <- request{value: value, op: OpSearch, result: res}
	return <-res
}

func (n *Node) Print() {
	fmt.Print(n.value)
	if n.left != nil {
		fmt.Print(" Left of ", n.value, ": ")
		n.left.Print()
	}
	if n.right != nil {
		fmt.Print(" Right of ", n.value, ": ")
		n.right.Print()
	}
}

func generateBinaryTreeOperations(valueRange int, op int, tree *FineGrainBinaryTree) {
	for i := 0; i < op; i++ {
		val := rand.IntN(valueRange) + 1
		switch method := opType(rand.IntN(3)); method {
		case OpInsert:
			fmt.Println("Start insert of ", val)
			tree.Insert(val)
		case OpRemove:
			fmt.Println("Start remove of ", val)
		case OpSearch:
			fmt.Println("Start search of ", val)
			tree.Search(val)
		}
	}
}

func RunBinaryTree(n int, valueRange int, op int) {
	fgbt := NewFineGrainBinaryTree()

	for i := 0; i < n; i++ {
		go generateBinaryTreeOperations(valueRange, op, fgbt)
	}
}
