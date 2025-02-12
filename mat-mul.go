package main

import (
	"math/rand/v2"
	"time"
)

type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

func generateMatrix[T Numeric](rows int, cols int) [][]T {
	matrix := make([][]T, rows)
	for i := range matrix {
		matrix[i] = make([]T, cols)
		for j := range matrix[i] {
			matrix[i][j] = T(rand.IntN(10))
		}
	}
	return matrix
}

func concurrentMatrixMultiplication[T Numeric](m1 [][]T, m2 [][]T, n int) ([][]T, time.Duration, float64, time.Duration) {
	if len(m1[0]) != len(m2) {
		return nil, 0, 0, 0
	}
	r1 := len(m1)
	c2 := len(m2[0])

	res := make([][]T, r1)
	resChannel := make(chan result[T], r1*c2)
	workChannel := make(chan position, r1*c2)
	latencyChannel := make(chan time.Duration, n)

	for i := range res {
		res[i] = make([]T, c2)

		for j := range c2 {
			workChannel <- position{i, j}
		}
	}

	startTime := time.Now()

	for i := 0; i < n; i++ {
		goroutineStartTime := time.Now()
		go calculateProducts(m1, m2, resChannel, workChannel, latencyChannel, goroutineStartTime)
	}

	for i := 0; i < r1*c2; i++ {
		r := <-resChannel
		res[r.pos.x][r.pos.y] = r.res
	}

	computationTime := time.Since(startTime)

	var totalLatency time.Duration
	for i := 0; i < n; i++ {
		totalLatency += <-latencyChannel
	}
	avgLatency := totalLatency / time.Duration(n)

	totalOperations := r1 * c2
	throughput := float64(totalOperations) / computationTime.Seconds()

	close(workChannel)
	return res, avgLatency, throughput, computationTime
}

func calculateProducts[T Numeric](m1 [][]T, m2 [][]T, resChannel chan result[T], workChannel chan position, latencyChannel chan time.Duration, t time.Time) {
	latencyChannel <- time.Since(t)
	for {
		pos, ok := <-workChannel
		if !ok {
			workChannel = nil
		}
		var col = make([]T, len(m2))
		for i := range len(m2) {
			col[i] = m2[i][pos.y]
		}
		dotProduct(m1[pos.x], col, pos.x, pos.y, resChannel)

		if workChannel == nil {
			break
		}
	}
}

func dotProduct[T Numeric](row []T, col []T, i int, j int, c chan result[T]) T {
	var product T
	for k := 0; k < len(row); k++ {
		product += row[k] * col[k]
	}
	c <- result[T]{product, position{i, j}}
	return product
}

type result[T Numeric] struct {
	res T
	pos position
}

type position struct {
	x int
	y int
}
