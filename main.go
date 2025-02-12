package main

import (
	"experiments/benchmarks"
)

func main() {
	g := 1
	benchmarks.RunParallelBinaryTree(g, 1000, 100)
}
