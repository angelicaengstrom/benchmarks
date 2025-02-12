package main

import (
	"fmt"
	"math/rand/v2"
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
}

type ConcurrentBinTree struct {
	requests chan Request
}

func NewConcurrentBinTree() *ConcurrentBinTree {
	cbt := &ConcurrentBinTree{
		requests: make(chan Request),
	}
	go cbt.run()
	return cbt
}

func (cbt *ConcurrentBinTree) Insert(val int) {
	cbt.requests <- Request{Op: OpInsert, Value: val}
}

func (cbt *ConcurrentBinTree) Remove(val int) {
	cbt.requests <- Request{Op: OpRemove, Value: val}
}

func (cbt *ConcurrentBinTree) Search(val int) bool {
	response := make(chan bool)
	cbt.requests <- Request{Op: OpSearch, Value: val, Result: response}
	return <-response
}

func (cbt *ConcurrentBinTree) run() {
	tree := BinTree{}
	for req := range cbt.requests {
		switch req.Op {
		case OpInsert:
			tree = *tree.Insert(req.Value)
			fmt.Println("\nInsert", req.Value)
			tree.Print()
		case OpRemove:
			tree = *tree.Remove(req.Value)
			fmt.Println("\nRemove", req.Value)
			tree.Print()
		case OpSearch:
			req.Result <- tree.Search(req.Value)
			fmt.Println("\nSearch", req.Value)
			tree.Print()
		}
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

func concurrentBinaryTree(n int, valueRange int, op int) {
	cbt := NewConcurrentBinTree()
	operations := make(chan OpType)

	for i := 0; i < n; i++ {
		go func() {
			for range op {
				method := rand.IntN(3)
				val := rand.IntN(valueRange) + 1
				switch method {
				case 0:
					cbt.Insert(val)
					operations <- OpInsert
				case 1:
					cbt.Remove(val)
					operations <- OpRemove
				case 2:
					cbt.Search(val)
					operations <- OpSearch
				}
			}
		}()
	}

	for i := 0; i < op; i++ {
		<-operations
	}
}
