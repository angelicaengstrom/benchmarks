package benchmarks

import (
	"fmt"
	"math/rand/v2"
	"time"
)

type ParallelBinaryTree struct {
	Val   int
	Left  *ParallelBinaryTree
	Right *ParallelBinaryTree
}

func (tree *ParallelBinaryTree) Insert(val int) *ParallelBinaryTree {
	if tree == nil || tree.Val == 0 {
		return &ParallelBinaryTree{Val: val}
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

func (tree *ParallelBinaryTree) Search(val int) *ParallelBinaryTree {
	if tree == nil || tree.Val == 0 {
		return tree
	}
	if tree.Val < val {
		return tree.Right.Search(val)
	} else if tree.Val > val {
		return tree.Left.Search(val)
	} else {
		return tree
	}
}

func (tree *ParallelBinaryTree) Remove(val int) *ParallelBinaryTree {
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

func (tree *ParallelBinaryTree) FindSuccessor() *ParallelBinaryTree {
	if tree.Left == nil {
		return tree
	}
	return tree.Left.FindSuccessor()
}

func (tree *ParallelBinaryTree) Print() {
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

func runOperations(op int, valueRange int, treeChannel chan *ParallelBinaryTree, resultChannel chan Metrics) {
	select {
	case tree := <-treeChannel:
		{
			val := rand.IntN(valueRange) + 1
			startTime := time.Now()
			switch method := OpType(rand.IntN(3)); method {
			case OpInsert:
				treeChannel <- tree.Insert(val)
				computationTime := time.Since(startTime)
				resultChannel <- Metrics{computationTime: computationTime}
			case OpRemove:
				treeChannel <- tree.Remove(val)
				computationTime := time.Since(startTime)
				resultChannel <- Metrics{computationTime: computationTime}
			case OpSearch:
				treeChannel <- tree.Search(val)
				computationTime := time.Since(startTime)
				resultChannel <- Metrics{computationTime: computationTime}
			}
		}
	}
}

func RunParallelBinaryTree(n int, valueRange int, op int) {
	treeChannel := make(chan *ParallelBinaryTree)
	resultChannel := make(chan Metrics)
	treeChannel <- &ParallelBinaryTree{}

	for i := 0; i < n; i++ {
		go runOperations(op, valueRange, treeChannel, resultChannel)
	}

	for i := 0; i < n; i++ {
		res := <-resultChannel
		print("Operation ", res.operation, " Result: ", res.computationTime)
	}
	tree := <-treeChannel
	tree.Print()
}
